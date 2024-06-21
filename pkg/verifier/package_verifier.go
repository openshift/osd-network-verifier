package verifier

import (
	"context"
	"time"

	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes"
	"github.com/openshift/osd-network-verifier/pkg/proxy"
)

// VerifierService defines the behaviors necessary to run verifier completely. Any clients that use that fullfills this interface
// will be able to run all verifier test
type verifierService interface {

	// ValidateEgress validates that all required targets are reachable from the vpcsubnet
	// target URLs: https://docs.openshift.com/rosa/rosa_getting_started/rosa-aws-prereqs.html#osd-aws-privatelink-firewall-prerequisites
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	ValidateEgress(vei ValidateEgressInput) *output.Output

	// VerifyDns verifies that a given VPC meets the DNS requirements specified in:
	// https://docs.openshift.com/container-platform/4.10/installing/installing_aws/installing-aws-vpc.html
	// Expected return value is *output.Output that's storing failures, exceptions and errors
	VerifyDns(vdi VerifyDnsInput) *output.Output
}

type ValidateEgressInput struct {
	Timeout                                            time.Duration
	Ctx                                                context.Context
	SubnetID, CloudImageID, InstanceType, PlatformType string
	Proxy                                              proxy.ProxyConfig
	Tags                                               map[string]string
	AWS                                                AwsEgressConfig
	GCP                                                GcpEgressConfig
	SkipInstanceTermination                            bool
	TerminateDebugInstance                             string
	ImportKeyPair                                      string
	ForceTempSecurityGroup                             bool
	Probe                                              probes.Probe
}
type AwsEgressConfig struct {
	KmsKeyID          string
	SecurityGroupId   string // Deprecated: prefer securityGroupIDs
	SecurityGroupIDs  []string
	TempSecurityGroup string
}
type GcpEgressConfig struct {
	Region, Zone, ProjectID, VpcName string
}

// ValidateEgress pass in a GCP or AWS client that know how to fufill above interface
func ValidateEgress(vs verifierService, vei ValidateEgressInput) *output.Output {
	return vs.ValidateEgress(vei)
}

type VerifyDnsInput struct {
	Ctx   context.Context
	VpcID string
}

// VerifyDns pass in a GCP or AWS client that know how to fufill above interface
func VerifyDns(vs verifierService, vdi VerifyDnsInput) *output.Output {
	return vs.VerifyDns(vdi)
}
