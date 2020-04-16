package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/majewsky/schwift"
	"github.com/sapcc/go-bits/httpee"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/image-migration-dashboard/internal/core"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // load the auth plugin
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var db core.Database

func fatalIfErr(err error) {
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func main() {
	inCluster := flag.Bool("in-cluster", false, "specify whether the application is running inside of k8s cluster")
	var kubeconfig *string
	if h := os.Getenv("HOME"); h != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(h, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	var (
		config *rest.Config
		err    error
	)
	if *inCluster {
		config, err = rest.InClusterConfig()
	} else {
		// use the current context in kubeconfig to build config
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}
	fatalIfErr(err)

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	fatalIfErr(err)

	// populate database using the backups from Swift
	db.DailyResults = make(map[string]core.ScanResult)
	db.LastScrapeTime = time.Now()

	acc, err := core.GetObjectStoreAccount()
	fatalIfErr(err)
	cntr, err := acc.Container(core.SwiftContainerName).EnsureExists()
	fatalIfErr(err)

	iter := cntr.Objects()
	err = iter.Foreach(func(o *schwift.Object) error {
		b, err := o.Download(nil).AsByteSlice()
		if err != nil {
			return err
		}

		if o.Name() == "image_data" {
			var data struct {
				Images core.ImageReport `json:"images"`
			}
			if err = json.Unmarshal(b, &data); err == nil {
				db.RW.Lock()
				db.Images = data.Images
				db.RW.Unlock()
			}
		} else {
			var data core.ScanResult
			if err = json.Unmarshal(b, &data); err == nil {
				db.RW.Lock()
				t := time.Unix(data.ScrapedAt, 0)
				db.LastScrapeTime = t
				db.DailyResults[t.Format(core.ISODateFormat)] = data
				db.RW.Unlock()
			}
		}

		return err
	})
	fatalIfErr(err)

	db.RW.Lock()
	if len(db.DailyResults)+
		len(db.Images.Keppel)+len(db.Images.Quay)+len(db.Images.Misc) > 0 {
		logg.Info("successfully populated the database from Swift backups")
	} else {
		logg.Info("could not populate the database from Swift since no data found")
	}
	db.RW.Unlock()

	go runCollector(&db, clientset)

	listenAddr := ":80"
	http.HandleFunc("/donut.png", handleGetDonutChart)
	http.HandleFunc("/graph.png", handleGetGraph)
	http.HandleFunc("/", handleHomePage)
	logg.Info("listening on " + listenAddr)
	err = httpee.ListenAndServeContext(httpee.ContextWithSIGINT(context.Background()), listenAddr, nil)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func runCollector(db *core.Database, clientset *kubernetes.Clientset) {
	ticker := time.Tick(30 * time.Minute)
	for range ticker {
		db.RW.RLock()
		t := db.LastScrapeTime
		db.RW.RUnlock()
		if time.Since(t) > 24*time.Hour {
			err := db.ScanCluster(clientset)
			if err != nil {
				logg.Error("cluster scan unsuccessful: %s", err.Error())
			}
		}
	}
}
