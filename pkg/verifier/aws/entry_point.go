package awsverifier

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"

	awsTools "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/openshift/osd-network-verifier/pkg/data/cloud"

	"github.com/openshift/osd-network-verifier/pkg/data/egress_lists"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/probes/curl"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
)

const (
	// ConfigPathFstring is the base path of the config file
	ConfigPathFstring = "/app/build/config/%s.yaml"
	DebugKeyName      = "onv-debug-key"
)

// ValidateEgress performs validation process for egress
// Basic workflow is:
// - prepare for ec2 instance creation
// - create instance and wait till it gets ready, wait for userdata script execution
// - find unreachable endpoints & parse output, then terminate instance
// - return `a.output` which stores the execution results
func (a *AwsVerifier) ValidateEgress(vei verifier.ValidateEgressInput) *output.Output {
	// Validate cloud platform type
	if !vei.PlatformType.IsValid() {
		vei.PlatformType = cloud.AWSClassic
	}
	// Default to curl.Probe if no Probe specified
	if vei.Probe == nil {
		vei.Probe = curl.Probe{}
		a.writeDebugLogs(vei.Ctx, "defaulted to curl probe")
	}

	// Default to 5sec per-request timeout if none specified
	if vei.Timeout <= 0 {
		vei.Timeout = 5 * time.Second
	}
	a.writeDebugLogs(vei.Ctx, fmt.Sprintf("configured a %s timeout for each egress request", vei.Timeout))

	var err error
	// Determine instance type and CPUArchitecture
	vei.InstanceType, vei.CPUArchitecture, err = a.selectInstanceType(vei.Ctx, vei.InstanceType, vei.CPUArchitecture)
	if err != nil {
		return a.Output.AddError(err)
	}

	// If no AMI specified, select one based on CPU arch and region
	if vei.CloudImageID == "" {
		vei.CloudImageID, err = vei.Probe.GetMachineImageID(vei.PlatformType, vei.CPUArchitecture, a.AwsClient.Region)
		if err != nil {
			return a.Output.AddError(fmt.Errorf("failed to determine default machine image: %w", err))
		}
		a.writeDebugLogs(vei.Ctx, fmt.Sprintf("defaulted to machine image %s", vei.CloudImageID))
	}

	// Select legacy probe config file based on platform type (ignored unless legacy.Probe in use)
	configPath := fmt.Sprintf(ConfigPathFstring, vei.PlatformType)

	var debugPubKey []byte
	// Check if Import-keypair flag has been passed
	if vei.ImportKeyPair != "" {
		// Read the pubkey file content into a variable
		PubKey, err := os.ReadFile(vei.ImportKeyPair)
		debugPubKey = PubKey
		if err != nil {
			return a.Output.AddError(err)
		}

		// Import Keypair into aws keypairs to be attached later to the created debug instance
		_, err = a.AwsClient.ImportKeyPair(vei.Ctx, &ec2.ImportKeyPairInput{
			KeyName:           awsTools.String(DebugKeyName),
			PublicKeyMaterial: debugPubKey,
		})
		if err != nil {
			return a.Output.AddError(err)
		}

		// If we have imported a pubkey for debug we would like debug instance to stay up.
		vei.SkipInstanceTermination = true

	}

	// Terminate a debug instance leftover from a previous run
	if vei.TerminateDebugInstance != "" {

		//Terminate the debug instance
		if err := a.AwsClient.TerminateEC2Instance(vei.Ctx, vei.TerminateDebugInstance); err != nil {
			a.Output.AddError(err)
		}

		// Check if a keypair was uploaded
		searchKeys := []string{DebugKeyName}
		_, err := a.AwsClient.DescribeKeyPairs(vei.Ctx, &ec2.DescribeKeyPairsInput{
			KeyNames: searchKeys,
		})
		if err != nil {
			// If no key was found continue without executing deletion code
			fmt.Printf("Debug KeyPair %v not found \n", DebugKeyName)
		} else {
			// If there was a key found, then delete it.
			_, err = a.AwsClient.DeleteKeyPair(vei.Ctx, &ec2.DeleteKeyPairInput{
				KeyName: awsTools.String(DebugKeyName),
			})
			// If there were any issues deleting the keypair.
			if err != nil {
				a.Output.AddError(err)
			}
		}

		return &a.Output
	}

	// Generate both egress lists for the given PlatformType. Note: the result of this is ignored by the Legacy probe.
	generatorVariables := map[string]string{"AWS_REGION": a.AwsClient.Region}
	generator := egress_lists.NewGenerator(vei.PlatformType, generatorVariables, a.Logger)

	egressListStr, tlsDisabledEgressListStr, err := generator.GenerateEgressLists(vei.Ctx, vei.EgressListYaml)
	if err != nil {
		return a.Output.AddError(err)
	}

	// Generate the userData file
	// As expand replaces all ${var} (using empty string for unknown ones), adding the env variables used in userdata.yaml
	if vei.Proxy.NoTls {
		a.Logger.Info(vei.Ctx, "NoTls enabled; ignoring any provided CA certs")
		vei.Proxy.Cacert = ""
	}

	userDataVariables := map[string]string{
		"AWS_REGION":       a.AwsClient.Region,
		"VALIDATOR_IMAGE":  networkValidatorImage,
		"VALIDATOR_REPO":   networkValidatorRepo,
		"TIMEOUT":          vei.Timeout.String(),
		"HTTP_PROXY":       vei.Proxy.HttpProxy,
		"HTTPS_PROXY":      vei.Proxy.HttpsProxy,
		"CACERT":           base64.StdEncoding.EncodeToString([]byte(vei.Proxy.Cacert)),
		"NO_PROXY":         vei.Proxy.NoProxyAsString(),
		"NOTLS":            strconv.FormatBool(vei.Proxy.NoTls),
		"CONFIG_PATH":      configPath,
		"DELAY":            "5",
		"URLS":             egressListStr,
		"TLSDISABLED_URLS": tlsDisabledEgressListStr,
	}

	if vei.SkipInstanceTermination {
		userDataVariables["DELAY"] = "60"
	}

	unencodedUserData, err := vei.Probe.GetExpandedUserData(userDataVariables)
	if err != nil {
		return a.Output.AddError(err)
	}
	unencodedUserDataBytes := []byte(unencodedUserData)
	// Enforce AWS-imposed userdata limit
	if len(unencodedUserDataBytes) > 16384 { // 16KB
		return a.Output.AddError(
			fmt.Errorf("userdata size exceeds AWS-imposed 16KB limit; if using a CA certificate, please check its file size"),
		)
	}
	userData := base64.StdEncoding.EncodeToString([]byte(unencodedUserData))

	a.writeDebugLogs(vei.Ctx, fmt.Sprintf("base64-encoded generated userdata script:\n---\n%s\n---", userData))

	vpcId, err := a.GetVpcIdFromSubnetId(vei.Ctx, vei.SubnetID)
	if err != nil {
		return a.Output.AddError(err)
	}

	// If security group not given, create a temporary one
	if len(vei.AWS.SecurityGroupIDs) == 0 || vei.ForceTempSecurityGroup {
		createSecurityGroupOutput, err := a.CreateSecurityGroup(vei.Ctx, vei.Tags, "osd-network-verifier", vpcId)
		if err != nil {
			return a.Output.AddError(err)
		}
		vei.AWS.TempSecurityGroup = *createSecurityGroupOutput.GroupId

		// Now that security group has been created, ensure we clean it up
		defer CleanupSecurityGroup(vei, a)

		// If proxy information given, add rules for it to the security group
		if vei.Proxy.HttpProxy != "" || vei.Proxy.HttpsProxy != "" {

			// Build a slice of proxy URLs (up to 2)
			proxyUrls := make([]string, 0, 2)
			if vei.Proxy.HttpProxy != "" {
				proxyUrls = append(proxyUrls, vei.Proxy.HttpProxy)
			}
			if vei.Proxy.HttpsProxy != "" {
				proxyUrls = append(proxyUrls, vei.Proxy.HttpsProxy)
			}

			// Add the new rules to the temp security group
			_, err := a.AllowSecurityGroupProxyEgress(vei.Ctx, vei.AWS.TempSecurityGroup, proxyUrls)
			if err != nil {
				return a.Output.AddError(err)
			}
		}
	}

	// Create EC2 instance
	instanceID, err := a.createEC2Instance(createEC2InstanceInput{
		amiID:               vei.CloudImageID,
		SubnetID:            vei.SubnetID,
		userdata:            userData,
		KmsKeyID:            vei.AWS.KmsKeyID,
		instanceCount:       instanceCount,
		ctx:                 vei.Ctx,
		instanceType:        vei.InstanceType,
		tags:                vei.Tags,
		securityGroupIDs:    vei.AWS.SecurityGroupIDs,
		tempSecurityGroupID: vei.AWS.TempSecurityGroup,
		keyPair:             vei.ImportKeyPair,
		vpcID:               vpcId,
	})
	if err != nil {
		return a.Output.AddError(err)
	}

	// ensurePrivate is a flag to ensure the return IP address from the given hosts are private defined in RFC1918
	// Currently, it will be used the Zero Egress cluster check only
	ensurePrivate := false
	if vei.PlatformType == cloud.AWSHCPZeroEgress {
		ensurePrivate = true
	}
	// findUnreachableEndpoints will call Probe.ParseProbeOutput(), which will store egress failures in a.Output.failures
	// when ensurePrivate is true, it will also check if the returned IP is private
	err = a.findUnreachableEndpoints(vei.Ctx, instanceID, vei.Probe, ensurePrivate)

	if err != nil {
		a.Output.AddError(err)
		// Don't return yet; still need to terminate instance
	}

	// Terminate the EC2 instance (unless user requests otherwise)
	if !vei.SkipInstanceTermination {
		// Replace the SecurityGroup attached to the instance with the default one for the VPC to allow for graceful
		// termination of the network-verifier created temporary SecurityGroup
		defaultSecurityGroupID := a.fetchVpcDefaultSecurityGroup(vei.Ctx, vpcId)
		if defaultSecurityGroupID != "" {
			err = a.modifyInstanceSecurityGroup(vei.Ctx, instanceID, defaultSecurityGroupID)
			if err != nil {
				a.Logger.Info(vei.Ctx, "Unable to detach instance from security group. Falling back to slower cloud resource cleanup method.")
				a.writeDebugLogs(vei.Ctx, fmt.Sprintf("Fell back to slower cloud resource cleanup because faster method (network interface detatchment) blocked by AWS: %s.", err))
			}
			a.Logger.Info(vei.Ctx, "Modified the instance to use the default security group")
		}

		a.Logger.Info(vei.Ctx, "Deleting instance with ID: %s", instanceID)
		if err := a.AwsClient.TerminateEC2Instance(vei.Ctx, instanceID); err != nil {
			a.Output.AddError(err)
		}
	}

	return &a.Output
}

// VerifyDns performs verification process for VPC's DNS
// Basic workflow is:
// - ask AWS API for VPC attributes
// - ensure they're set correctly
func (a *AwsVerifier) VerifyDns(vdi verifier.VerifyDnsInput) *output.Output {
	a.Logger.Info(vdi.Ctx, "Verifying DNS config for VPC %s", vdi.VpcID)
	// Request boolean values from AWS API
	dnsSprtResult, err := a.AwsClient.DescribeVpcAttribute(vdi.Ctx, &ec2.DescribeVpcAttributeInput{
		Attribute: ec2Types.VpcAttributeNameEnableDnsSupport,
		VpcId:     awsTools.String(vdi.VpcID),
	})
	if err != nil {
		a.Output.AddError(handledErrors.NewGenericError(err))
		a.Output.AddException(handledErrors.NewGenericError(
			fmt.Errorf("failed to validate the %s attribute on VPC: %s is true", ec2Types.VpcAttributeNameEnableDnsSupport, vdi.VpcID)),
		)
		return &a.Output
	}

	dnsHostResult, err := a.AwsClient.DescribeVpcAttribute(vdi.Ctx, &ec2.DescribeVpcAttributeInput{
		Attribute: "enableDnsHostnames",
		VpcId:     awsTools.String(vdi.VpcID),
	})
	if err != nil {
		a.Output.AddError(handledErrors.NewGenericError(err))
		a.Output.AddException(handledErrors.NewGenericError(
			fmt.Errorf("failed to validate the %s attribute on VPC: %s is true", ec2Types.VpcAttributeNameEnableDnsHostnames, vdi.VpcID),
		))
		return &a.Output
	}
	// Verify results
	a.Logger.Info(vdi.Ctx, "DNS Support for VPC %s: %t", vdi.VpcID, *dnsSprtResult.EnableDnsSupport.Value)
	a.Logger.Info(vdi.Ctx, "DNS Hostnames for VPC %s: %t", vdi.VpcID, *dnsHostResult.EnableDnsHostnames.Value)
	if !(*dnsSprtResult.EnableDnsSupport.Value) {
		a.Output.AddException(handledErrors.NewGenericError(
			fmt.Errorf("the %s attribute on VPC: %s is %t, must be true", ec2Types.VpcAttributeNameEnableDnsSupport, vdi.VpcID, *dnsSprtResult.EnableDnsSupport.Value),
		))
	}

	if !(*dnsHostResult.EnableDnsHostnames.Value) {
		a.Output.AddException(handledErrors.NewGenericError(
			fmt.Errorf("the %s attribute on VPC: %s is %t, must be true", ec2Types.VpcAttributeNameEnableDnsHostnames, vdi.VpcID, *dnsHostResult.EnableDnsHostnames.Value),
		))
	}

	return &a.Output
}

// CleanupSecurityGroup cleans up the security groups created by network-verifier
func CleanupSecurityGroup(vei verifier.ValidateEgressInput, a *AwsVerifier) *output.Output {
	a.Logger.Info(vei.Ctx, "Deleting security group with ID: %s", vei.AWS.TempSecurityGroup)
	_, err := a.AwsClient.DeleteSecurityGroup(vei.Ctx, &ec2.DeleteSecurityGroupInput{GroupId: awsTools.String(vei.AWS.TempSecurityGroup)})
	if err != nil {
		a.Output.AddError(handledErrors.NewGenericError(err))
		a.Output.AddException(handledErrors.NewGenericError(fmt.Errorf("unable to cleanup security group %s, please manually clean up", vei.AWS.TempSecurityGroup)))

	}
	return &a.Output
}
