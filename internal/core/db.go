package core

import (
	"sync"
	"time"
)

// ISODateFormat is what it is.
const ISODateFormat = "2006-01-02"

// Database is the in-memory database that persists for the duration of the
// application execution. It holds the required data to render the dashboard.
type Database struct {
	RW             sync.RWMutex
	DailyResults   map[string]ScanResult // map of date (ISO format) to ScanResult
	Images         ImageReport
	LastScrapeTime time.Time
}

// ScanResult holds the processed data for a single cluster scan.
type ScanResult struct {
	ScrapedAt  int64 `json:"scraped_at"` // UTC
	NoOfImages struct {
		Total  int `json:"total"`
		Keppel int `json:"keppel"`
		// Note: here Quay refers to the self-hosted Quay, not the public Quay.io
		Quay int `json:"quay"`
		Misc int `json:"misc"`
	} `json:"no_of_images"`
}

// ImageReport holds the data for all the images.
type ImageReport struct {
	Keppel []Image `json:"keppel"`
	Quay   []Image `json:"quay"`
	Misc   []Image `json:"misc"`
}

// Image holds the data for a specific image.
type Image struct {
	Name string `json:"name"`
	// Container names are in the form: namespace/pod/container
	Containers []string `json:"containers"`
}
