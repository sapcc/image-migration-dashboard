package core

import (
	"sync"
	"time"
)

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
		Total  uint64 `json:"total"`
		Keppel uint64 `json:"keppel"`
		// Note: here Quay refers to the self-hosted Quay, not the public Quay.io
		Quay uint64 `json:"quay"`
		Misc uint64 `json:"misc"`
	} `json:"no_of_images"`
}

// ImageData is a map of image name to pod/containers that are using it.
type ImageData map[string][]string
