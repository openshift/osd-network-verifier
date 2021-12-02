package cloudclient

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
)

// CloudClient defines the interface for a cloud agnostic implementation
// For mocking: mockgen -source=pkg/cloudclient/cloudclient.go -destination=pkg/cloudclient/mock_cloudclient/mock_cloudclient.go
type CloudClient interface {

	// ByoVPCValidator validates the configuration given by the customer
	ByoVPCValidator(ctx context.Context) error

	// ValidateEgress validates that all required targets are reachable from the vpcsubnet
	// required target are defined in https://docs.openshift.com/rosa/rosa_getting_started/rosa-aws-prereqs.html#osd-aws-privatelink-firewall-prerequisites
	ValidateEgress(ctx context.Context, vpcSubnetID, cloudImageID string) error
}

var controllerMapping = map[configv1.PlatformType]Factory{}

type Factory func() CloudClient

func Register(name configv1.PlatformType, factoryFunc Factory) {
	controllerMapping[name] = factoryFunc
}

// GetClientFor returns the CloudClient for the given cloud provider, identified
// by the provider's ID, eg aws for AWS's cloud client, gcp for GCP's cloud
// client.
func GetClientFor(cloudID configv1.PlatformType) CloudClient {
	if _, ok := controllerMapping[cloudID]; ok {
		return controllerMapping[cloudID]()
	}
	// TODO: Return a minimal interface?
	panic(fmt.Sprintf("Couldn't find a client matching %s", cloudID))
}
