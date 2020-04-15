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
