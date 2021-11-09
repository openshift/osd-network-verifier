package gcp

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier configv1.PlatformType = configv1.GCPPlatformType

// Client represents a GCP Client
type Client struct {
	projectID      string
	computeService *computev1.Service
}

func (c *Client) ByoVPCValidator(context.Context) error {
	return nil
}

// assumes serviceAccountJSON is being used. Feel free to change as needed
func NewClient() (*Client, error) {
	ctx := context.Background()
	var tempServiceAccountJSON []byte // to be implemented.

	// initialize actual client
	return newClient(ctx, tempServiceAccountJSON)
}

func newClient(ctx context.Context, serviceAccountJSON []byte) (*Client, error) {
	credentials, err := google.CredentialsFromJSON(
		ctx, serviceAccountJSON,
		computev1.ComputeScope)
	if err != nil {
		return nil, err
	}

	computeService, err := computev1.NewService(ctx, option.WithCredentials(credentials))
	if err != nil {
		return nil, err
	}

	return &Client{
		projectID:      credentials.ProjectID,
		computeService: computeService,
	}, nil
}
