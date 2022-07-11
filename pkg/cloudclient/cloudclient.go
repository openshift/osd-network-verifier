package cloudclient

import (
	"context"
	"fmt"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/utils"
)

// common commandline args
type CmdOptions struct {
	// AWS client options
	KmsKeyID        string
	CloudImageID    string
	Region          string
	InstanceType    string
	CloudTags       map[string]string
	AccessKeyId     string
	SessionToken    string
	SecretAccessKey string
	AwsProfile      string

	// GCP options
	// todo

	// Operation options
	Debug     bool
	Timeout   time.Duration
	CloudType string

	// Following are passed to client to mitigate "cannot create context from nil parent" error
	Ctx    context.Context
	Logger ocmlog.Logger
}

var (
	DefaultTime = 2 * time.Second
	Debug       = false
)

// CloudClient defines the interface for a cloud agnostic implementation
// For mocking: mockgen -source=pkg/cloudclient/cloudclient.go -package mocks -destination=pkg/cloudclient/mocks/mock_cloudclient.go
type CloudClient interface {

	// ByoVPCValidator validates the configuration given by the customer
	ByoVPCValidator(params ValidateByoVpc) error

	// ValidateEgress validates that all required targets are reachable from the vpcsubnet
	// target URLs: https://docs.openshift.com/rosa/rosa_getting_started/rosa-aws-prereqs.html#osd-aws-privatelink-firewall-prerequisites
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	ValidateEgress(params ValidateEgress) *output.Output

	// VerifyDns verifies that a given VPC meets the DNS requirements specified in:
	// https://docs.openshift.com/container-platform/4.10/installing/installing_aws/installing-aws-vpc.html
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	VerifyDns(params ValidateDns) *output.Output
}

var controllerMapping = map[string]Factory{}

type Factory func(options *CmdOptions) (CloudClient, error)

func Register(providerType string, factoryFunc Factory) {
	controllerMapping[providerType] = factoryFunc
}

// GetClientFor returns the CloudClient for any cloud provider

func GetClientFor(options *CmdOptions) (CloudClient, error) {
	platformType := utils.PlatformType(options.CloudType)
	//if _, ok := controllerMapping[platformType]; ok {
	cli, err := controllerMapping[platformType](options)
	//}
	if err != nil {
		return nil, (fmt.Errorf("Couldn't create cloud client for %s: %s", platformType, err))

	}
	return cli, nil
}

type ValidateEgress struct {
	VpcSubnetID string
}

type ValidateDns struct {
	VpcId string
}

type ValidateByoVpc struct {
	//	todo
}
