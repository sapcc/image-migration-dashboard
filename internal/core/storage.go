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
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/majewsky/schwift"
	"github.com/majewsky/schwift/gopherschwift"
)

// SwiftContainerName is the name of the Swift container where ScanResult
// backups are stored.
const (
	SwiftContainerName = "image-migration-dashboard"
	ScanResultPrefix   = "scan-result"
)

// GetObjectStoreAccount logs in to an OpenStack cloud, acquires a token, and
// returns the relevant Swift account.
func GetObjectStoreAccount() (*schwift.Account, error) {
	provider, err := clientconfig.AuthenticatedClient(nil)
	if err != nil {
		return nil, fmt.Errorf("could not initialize openstack client: %s", err.Error())
	}
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("could not initialize object storage client: %s", err.Error())
	}

	account, err := gopherschwift.Wrap(client, nil)
	if err != nil {
		return nil, err
	}

	return account, nil
}
