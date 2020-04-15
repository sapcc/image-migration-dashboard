package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/sapcc/go-bits/logg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var imageFormatRx = regexp.MustCompile(`^(\S+)\.\S+\.\S+\.\S+\/.*$`)

// ScanCluster scans a cluster for all the pods, processes the information,
// and saves it to the object store.
func (db *Database) ScanCluster(clientset *kubernetes.Clientset) error {
	now := time.Now()
	date := now.Format(ISODateFormat)

	// get pods in all the namespaces
	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	logg.Info("%d pods scanned", len(pods.Items))

	var result ScanResult
	result.ScrapedAt = now.Unix()

	// get all images
	allImgs := make(map[string][]string)
	for _, pod := range pods.Items {
		ns := pod.ObjectMeta.GetNamespace()
		podName := pod.ObjectMeta.GetName()
		for _, c := range pod.Spec.Containers {
			n := fmt.Sprintf("%s/%s/%s", ns, podName, c.Name)
			allImgs[c.Image] = append(allImgs[c.Image], n)
		}
		for _, c := range pod.Spec.InitContainers {
			n := fmt.Sprintf("%s/%s/%s", ns, podName, c.Name)
			allImgs[c.Image] = append(allImgs[c.Image], n)
		}
	}

	// determine image registry and sort the date alphabetically
	keys := make([]string, 0, len(allImgs))
	for k := range allImgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var imgReport ImageReport
	for _, v := range keys {
		cntrs := allImgs[v]
		sort.Strings(cntrs)

		matchList := imageFormatRx.FindStringSubmatch(v)
		if matchList != nil {
			switch matchList[1] {
			case "keppel":
				imgReport.Keppel = append(imgReport.Keppel, Image{Name: v, Containers: cntrs})
			case "hub":
				imgReport.Quay = append(imgReport.Quay, Image{Name: v, Containers: cntrs})
			default:
				imgReport.Misc = append(imgReport.Misc, Image{Name: v, Containers: cntrs})
			}
		} else {
			imgReport.Misc = append(imgReport.Misc, Image{Name: v, Containers: cntrs})
		}
	}
	result.NoOfImages.Keppel = len(imgReport.Keppel)
	result.NoOfImages.Quay = len(imgReport.Quay)
	result.NoOfImages.Misc = len(imgReport.Misc)
	result.NoOfImages.Total = result.NoOfImages.Keppel + result.NoOfImages.Quay + result.NoOfImages.Misc
	logg.Info("%d images found: %d from Keppel, %d from Quay, and %d from misc. sources",
		result.NoOfImages.Total, result.NoOfImages.Keppel,
		result.NoOfImages.Quay, result.NoOfImages.Misc)

	db.RW.Lock()
	db.DailyResults[date] = result
	db.Images = imgReport
	db.LastScrapeTime = now
	db.RW.Unlock()
	logg.Info("successfully updated the database")

	// upload ScanResult and images data to Swift
	acc, err := GetObjectStoreAccount()
	if err != nil {
		return err
	}
	cntr, err := acc.Container(SwiftContainerName).EnsureExists()
	if err != nil {
		return err
	}

	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	n := filepath.Join(ScanResultPrefix, date)
	obj := cntr.Object(n)
	err = obj.Upload(bytes.NewReader(b), nil, nil)
	if err != nil {
		return err
	}
	logg.Info("uploaded scan result to %s", obj.FullName())

	b, err = json.Marshal(struct {
		Images ImageReport `json:"images"`
	}{imgReport})
	if err != nil {
		return err
	}
	obj = cntr.Object("image_data")
	err = obj.Upload(bytes.NewReader(b), nil, nil)
	if err != nil {
		return err
	}
	logg.Info("uploaded image data to %s", obj.FullName())

	return nil
}
