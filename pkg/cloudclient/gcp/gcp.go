package gcp

import (
	"context"
	"fmt"

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
	fmt.Println("interface executed: " + ClientIdentifier)
	return nil
}

// assumes serviceAccountJSON is being used. Feel free to change as needed
func NewClient() (*Client, error) {
	ctx := context.Background()
	dummySA := `{
		"type": "service_account",
		"private_key_id": "abc",
		"private_key": "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----\n",
		"client_email": "123-abc@developer.gserviceaccount.com",
		"client_id": "123-abc.apps.googleusercontent.com",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "http://localhost:8080/token"
	  }`

	// initialize actual client
	return newClient(ctx, []byte(dummySA))
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
