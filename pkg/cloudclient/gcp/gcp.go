package gcp

import (
	"context"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier = "GCP"

// Client represents a GCP Client
type Client struct {
	projectID      string
	region         string
	instanceType   string
	computeService *computev1.Service
	tags           map[string]string
	logger         ocmlog.Logger
	output         output.Output
}

type ClientInput struct {
	Ctx          context.Context
	Logger       ocmlog.Logger
	Creds        *google.Credentials
	Region       string
	InstanceType string
	Tags         map[string]string
	Timeout      time.Duration
}

func (c *Client) ByoVPCValidator(ctx context.Context) error {
	c.logger.Info(ctx, "interface executed: %s", ClientIdentifier)
	return nil
}

func (c *Client) ValidateEgress(ctx context.Context) *output.Output {
	return &c.output
}

func (c *Client) VerifyDns(ctx context.Context, vpcID string) *output.Output {
	return &c.output
}

func NewClient(ctx context.Context, logger ocmlog.Logger, credentials *google.Credentials, region, instanceType string, tags map[string]string) (*Client, error) {
	// initialize actual client
	// todo implement credentials = getGcpCredsFromInput()
	clientInput := &ClientInput{
		Ctx:          ctx,
		Logger:       logger,
		Region:       region,
		InstanceType: instanceType,
		Tags:         tags,
		Creds:        credentials,
	}
	return newClient(*clientInput)
}

func newClient(input ClientInput) (*Client, error) {
	computeService, err := computev1.NewService(input.Ctx, option.WithCredentials(input.Creds))
	if err != nil {
		return nil, err
	}

	return &Client{
		projectID:      input.Creds.ProjectID,
		region:         input.Region,
		instanceType:   input.InstanceType,
		computeService: computeService,
		tags:           input.Tags,
		logger:         input.Logger,
	}, nil
}
