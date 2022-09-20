package gcp

import (
	"context"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
	"golang.org/x/oauth2/google"
	computev1 "google.golang.org/api/compute/v1"
)

// ClientIdentifier is what kind of cloud this implement supports
const ClientIdentifier string = "GCP"

// Client represents a GCP Client
type Client struct {
	projectID      string
	region         string
	zone           string
	instanceType   string
	computeService *computev1.Service
	tags           map[string]string
	logger         ocmlog.Logger
	output         output.Output
}

func (c *Client) ByoVPCValidator(ctx context.Context) error {
	c.logger.Info(ctx, "interface executed: %s", ClientIdentifier)
	return nil
}

func (c *Client) ValidateEgress(ctx context.Context, vpcSubnetID, cloudImageID, kmsKeyID, securityGroupId string, timeout time.Duration, proxy proxy.ProxyConfig) *output.Output {
	return c.validateEgress(ctx, vpcSubnetID, cloudImageID, kmsKeyID, timeout, proxy)
}

func (c *Client) VerifyDns(ctx context.Context, vpcID string) *output.Output {
	return &c.output
}

func NewClient(ctx context.Context, logger ocmlog.Logger, credentials *google.Credentials, region, instanceType string, tags map[string]string) (*Client, error) {
	// initialize actual client
	return newClient(ctx, logger, credentials, region, instanceType, tags)
}
