package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
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
	date := now.Format("2006-01-02") // ISO format

	// get pods in all the namespaces
	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	logg.Info("%d pods scanned", len(pods.Items))

	var result ScanResult
	result.ScrapedAt = now.Unix()

	// sort containers to their images
	tmp := make(ImageData)
	for _, pod := range pods.Items {
		podName := pod.ObjectMeta.GetName()
		for _, c := range pod.Spec.Containers {
			n := fmt.Sprintf("%s/%s", podName, c.Name)
			tmp[c.Image] = append(tmp[c.Image], n)
		}
		for _, c := range pod.Spec.InitContainers {
			n := fmt.Sprintf("%s/%s", podName, c.Name)
			tmp[c.Image] = append(tmp[c.Image], n)
		}
	}

	// determine image source and discard misc. image data
	// we are only interested in containers using Keppel or Quay
	images := make(ImageData)
	for name, cntrs := range tmp {
		matchList := imageFormatRx.FindStringSubmatch(name)
		if matchList != nil {
			switch matchList[1] {
			case "keppel":
				result.NoOfImages.Keppel++
			case "hub":
				result.NoOfImages.Quay++
			}
			images[name] = cntrs
		} else {
			result.NoOfImages.Misc++
		}
	}
	result.NoOfImages.Total = result.NoOfImages.Keppel + result.NoOfImages.Quay + result.NoOfImages.Misc
	logg.Info("%d images found: %d from Keppel, %d from Quay, and %d from misc. sources",
		result.NoOfImages.Total, result.NoOfImages.Keppel,
		result.NoOfImages.Quay, result.NoOfImages.Misc)

	db.RW.Lock()
	db.DailyResults[date] = result
	db.Images = images
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
		Images ImageData `json:"images"`
	}{images})
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
