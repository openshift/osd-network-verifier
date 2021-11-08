package gcp

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	computev1 "google.golang.org/api/compute/v1"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier configv1.PlatformType = configv1.GCPPlatformType

// Client represents a GCP Client
type Client struct {
	projectID      string
	region         string
	clusterName    string
	baseDomain     string
	computeService *computev1.Service
}

func (c *Client) ByoVPCValidator(context.Context) error {
	return nil
}
