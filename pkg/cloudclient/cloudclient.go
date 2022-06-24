package cloudclient

import (
	"context"
	"fmt"
	"os"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	awsCloudClient "github.com/openshift/osd-network-verifier/pkg/cloudclient/aws"
	gcpCloudClient "github.com/openshift/osd-network-verifier/pkg/cloudclient/gcp"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"golang.org/x/oauth2/google"
)

// common commandline args
type CmdOptions struct {
	CloudType    string // not required. if provided, currently only supports "aws".
	CloudTags    map[string]string
	Debug        bool
	Region       string
	AwsProfile   string
	InstanceType string
}

// CloudClient defines the interface for a cloud agnostic implementation
// For mocking: mockgen -source=pkg/cloudclient/cloudclient.go -package mocks -destination=pkg/cloudclient/mocks/mock_cloudclient.go
type CloudClient interface {

	// ByoVPCValidator validates the configuration given by the customer
	ByoVPCValidator(ctx context.Context) error

	// ValidateEgress validates that all required targets are reachable from the vpcsubnet
	// target URLs: https://docs.openshift.com/rosa/rosa_getting_started/rosa-aws-prereqs.html#osd-aws-privatelink-firewall-prerequisites
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	ValidateEgress(ctx context.Context) *output.Output

	// VerifyDns verifies that a given VPC meets the DNS requirements specified in:
	// https://docs.openshift.com/container-platform/4.10/installing/installing_aws/installing-aws-vpc.html
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	VerifyDns(ctx context.Context, vpcID string) *output.Output
}

func NewClient(ctx context.Context, logger ocmlog.Logger,
	options CmdOptions) (CloudClient, error) {
	switch options.CloudType {

	case "aws":
		clientInput := &awsCloudClient.ClientInput{
			Ctx:             ctx,
			Logger:          logger,
			Region:          options.Region,
			InstanceType:    options.InstanceType,
			Tags:            options.CloudTags,
			Profile:         options.AwsProfile,
			AccessKeyId:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
			VpcSubnetID:     ctx.Value("VpcSubnetID").(string),
			CloudImageID:    ctx.Value("CloudImageID").(string),
			Timeout:         ctx.Value("Timeout").(time.Duration),
			KmsKeyID:        ctx.Value("KmsKeyID").(string),
		}
		return awsCloudClient.NewClient(clientInput)
	case "gcp":
		var gcpCreds *google.Credentials //todo remove gcpCreds arg once getGcpCredsFromInput is implemented in GCP NewClient
		return gcpCloudClient.NewClient(ctx, logger, gcpCreds, options.Region, options.InstanceType, options.CloudTags)
	default:
		return nil, fmt.Errorf("unsupported cloud client type")
	}

}
