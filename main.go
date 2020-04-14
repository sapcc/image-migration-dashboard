package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/majewsky/schwift"
	"github.com/sapcc/go-bits/httpee"
	"github.com/sapcc/go-bits/logg"
	"github.com/sapcc/image-migration-dashboard/internal/core"
	"github.com/wcharczuk/go-chart"

	// load the auth plugin

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/tools/clientcmd"
)

var db core.Database

func fatalIfErr(err error) {
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func main() {
	// get path to the kubeconfig file
	var kubeconfig *string
	if h := os.Getenv("HOME"); h != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(h, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig to build config
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	fatalIfErr(err)

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	fatalIfErr(err)

	// populate database using the backups from Swift
	db.DailyResults = make(map[string]core.ScanResult)
	db.Images = make(core.ImageData)

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
				Images core.ImageData `json:"images"`
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
				db.DailyResults[t.Format("2006-01-02")] = data
				db.RW.Unlock()
			}
		}

		return err
	})
	fatalIfErr(err)

	db.RW.Lock()
	if len(db.DailyResults)+len(db.Images) > 0 {
		logg.Info("successfully populated the database from Swift backups")
	} else {
		logg.Info("could not populate the database from Swift since no data found")
	}
	db.RW.Unlock()

	go runCollector(&db, clientset)

	listenAddr := ":8080"
	http.HandleFunc("/donut.png", handleGetDonutChart)
	http.HandleFunc("/graph.png", handleGetGraph)
	http.HandleFunc("/", homepageHandler)
	logg.Info("listening on " + listenAddr)
	err = httpee.ListenAndServeContext(httpee.ContextWithSIGINT(context.Background()), listenAddr, nil)
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func runCollector(db *core.Database, clientset *kubernetes.Clientset) {
	ticker := time.Tick(30 * time.Minute)
	for {
		select {
		case <-ticker:
			db.RW.RLock()
			t := db.LastScrapeTime
			db.RW.RUnlock()
			if time.Since(t) > 24*time.Hour {
				err := db.ScanCluster(clientset)
				if err != nil {
					logg.Error(err.Error())
				}
			}
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
// http.HandleFunc(s)

const homePageHTML = `<!DOCTYPE html>
<html lang="en-us">
<head>
	<meta charset="utf-8">
  <meta http-equiv="x-ua-compatible" content="ie=edge">
	<title>Image Migration Dashboard</title>
	<meta name="description" content="Image Migration Dashboard">
  <meta name="viewport" content="width=device-width, initial-scale=1">
	<style type="text/css" media="screen">body{width:auto;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;font-size:1.1rem;font-weight:400;line-height:1.5;color:#212529;background-color:#fff}h1{font-size:2.5em}h3{font-size:1.25em}.heading{margin-top:2em}</style>
</head>
<body>
<div class="container">
	<section class="row heading">
		<h1>Image Migration Dashboard</h1>
	</section>
	<section class="row">
		<h3>Image usage over time</h3>
		<img src="/graph.png">
	</section>
	<section class="row">
		<h3>Current image distribution</h3>
		<p>Number of Images:</p>
		<ul>
			<li>Total: {{ .LastResult.NoOfImages.Total }}</li>
			<li>Keppel: {{ .LastResult.NoOfImages.Keppel }}</li>
			<li>Quay: {{ .LastResult.NoOfImages.Quay }}</li>
			<li>Misc.: {{ .LastResult.NoOfImages.Misc }}</li>
		</ul>
		<br>
		<img src="/donut.png">
	</section>
	<section class="row">
		<h3>Images</h3>
		<table>
		<tr>
			<th>Image</th>
			<th>Pod/Container</th>
		</tr>
		{{ range $k, $v := .Images }}
		<tr>
			<td>{{ $k }}</td>
			<td>{{ $v }}</td>
		</tr>
		{{ end }}
		</table>
	</section>
</div>
</body>
</html>`

func homepageHandler(w http.ResponseWriter, r *http.Request) {
	db.RW.RLock()
	res := db.DailyResults[db.LastScrapeTime.Format("2006-01-02")]
	images := db.Images
	db.RW.RUnlock()

	data := struct {
		LastResult core.ScanResult
		Images     core.ImageData
	}{res, images}

	t, err := template.New("homepage").Parse(homePageHTML)
	if err != nil {
		logg.Error(err.Error())
	}

	t.Execute(w, data)
}

// HandleGetDonutChart serves donuts.
func handleGetDonutChart(w http.ResponseWriter, r *http.Request) {
	db.RW.RLock()
	res := db.DailyResults[db.LastScrapeTime.Format("2006-01-02")]
	db.RW.RUnlock()

	donut := chart.DonutChart{
		Width:  512,
		Height: 512,
		Values: []chart.Value{
			{Value: float64(res.NoOfImages.Keppel), Label: "Keppel"},
			{Value: float64(res.NoOfImages.Quay), Label: "Quay"},
			{Value: float64(res.NoOfImages.Misc), Label: "Misc"},
		},
	}
	var b bytes.Buffer
	err := donut.Render(chart.PNG, &b)
	if err != nil {
		logg.Error(err.Error())
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(b.Bytes())
}

// HandleGetGraph serves the graph.
func handleGetGraph(w http.ResponseWriter, r *http.Request) {
	db.RW.RLock()
	ts := []time.Time{}
	kfs := []float64{}
	qfs := []float64{}
	for _, v := range db.DailyResults {
		ts = append(ts, time.Unix(v.ScrapedAt, 0))
		kfs = append(kfs, float64(v.NoOfImages.Keppel))
		qfs = append(qfs, float64(v.NoOfImages.Quay))
	}
	db.RW.RUnlock()

	graph := chart.Chart{
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   60,
				Right:  20,
				Bottom: 20,
			},
		},
		Series: []chart.Series{
			chart.TimeSeries{
				Name:    "Keppel",
				XValues: ts,
				YValues: kfs,
			},
			chart.TimeSeries{
				Name:    "Quay",
				XValues: ts,
				YValues: qfs,
			},
		},
	}
	//note we have to do this as a separate step because we need a reference to graph
	graph.Elements = []chart.Renderable{
		chart.LegendLeft(&graph),
	}
	var b bytes.Buffer
	err := graph.Render(chart.PNG, &b)
	if err != nil {
		logg.Error(err.Error())
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(b.Bytes())
}
