// Copyright 2020 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		Quay      int `json:"quay"`
		DockerHub int `json:"docker_hub"`
		Misc      int `json:"misc"`
	} `json:"no_of_images"`
}

// ImageReport holds the data for all the images.
type ImageReport struct {
	Keppel    []Image `json:"keppel"`
	Quay      []Image `json:"quay"`
	DockerHub []Image `json:"docker_hub"`
	Misc      []Image `json:"misc"`
}

// Image holds the data for a specific image.
type Image struct {
	Name string `json:"name"`
	// Container names are in the form: namespace/pod/container
	Containers []string `json:"containers"`
}
