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
	Images         ImageData
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

// ImageData is a map of registry name to a map of image name to container
// names that are using it.
// Containers names are in the form: namespace/pod/container
type ImageData map[string]map[string][]string
