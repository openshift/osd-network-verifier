package awsverifier

import (
	"encoding/base64"
	"fmt"
	"strconv"

	awsTools "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	handledErrors "github.com/openshift/osd-network-verifier/pkg/errors"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/output"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
)

// Base path of the config file
const CONFIG_PATH_FSTRING string = "/app/build/config/%s.yaml"

// ValidateEgress performs validation process for egress
// Basic workflow is:
// - prepare for ec2 instance creation
// - create instance and wait till it gets ready, wait for userdata script execution
// - find unreachable endpoints & parse output, then terminate instance
// - return `a.output` which stores the execution results
func (a *AwsVerifier) ValidateEgress(vei verifier.ValidateEgressInput) *output.Output {
	a.writeDebugLogs(fmt.Sprintf("Using configured timeout of %s for each egress request", vei.Timeout.String()))

	// Set default instance type if non is found
	if vei.InstanceType == "" {
		vei.InstanceType = "t3.micro"
	}

	// Validates the provided instance type will work with the verifier
	// NOTE a "nitro" EC2 instance type is required to be used
	if err := a.validateInstanceType(vei.Ctx, vei.InstanceType); err != nil {
		return a.Output.AddError(fmt.Errorf("instance type %s is invalid: %s", vei.InstanceType, err))
	}

	// Select config file based on platform type
	configPath := fmt.Sprintf(CONFIG_PATH_FSTRING, vei.PlatformType)
	if vei.PlatformType == "" {
		// Default to AWS
		configPath = fmt.Sprintf(CONFIG_PATH_FSTRING, helpers.PlatformAWS)
	}

	// Terminate a debug instance leftover from a previous run
	if vei.TerminateDebugInstance != "" {
		if err := a.AwsClient.TerminateEC2Instance(vei.Ctx, vei.TerminateDebugInstance); err != nil {
			a.Output.AddError(err)
		}
		return &a.Output
	}

	// Generate the userData file
	// As expand replaces all ${var} (using empty string for unknown ones), adding the env variables used in userdata.yaml
	userDataVariables := map[string]string{
		"AWS_REGION":               a.AwsClient.Region,
		"USERDATA_BEGIN":           "USERDATA BEGIN",
		"USERDATA_END":             userdataEndVerifier,
		"VALIDATOR_START_VERIFIER": "VALIDATOR START",
		"VALIDATOR_END_VERIFIER":   "VALIDATOR END",
		"VALIDATOR_IMAGE":          networkValidatorImage,
		"TIMEOUT":                  vei.Timeout.String(),
		"HTTP_PROXY":               vei.Proxy.HttpProxy,
		"HTTPS_PROXY":              vei.Proxy.HttpsProxy,
		"CACERT":                   base64.StdEncoding.EncodeToString([]byte(vei.Proxy.Cacert)),
		"NOTLS":                    strconv.FormatBool(vei.Proxy.NoTls),
		"IMAGE":                    "$IMAGE",
		"VALIDATOR_REFERENCE":      "$VALIDATOR_REFERENCE",
		"CONFIG_PATH":              configPath,
		"DELAY":                    "5",
	}

	if vei.SkipInstanceTermination {
		userDataVariables["DELAY"] = "60"
	}

	userData, err := generateUserData(userDataVariables)
	if err != nil {
		return a.Output.AddError(err)
	}
	a.writeDebugLogs(fmt.Sprintf("base64-encoded generated userdata script:\n---\n%s\n---", userData))

	err = setCloudImage(&vei.CloudImageID, a.AwsClient.Region)
	if err != nil {
		return a.Output.AddError(err) // fatal
	}

	cleanupSecurityGroup := false
	if vei.AWS.SecurityGroupId == "" {
		vpcId, err := a.GetVpcIdFromSubnetId(vei.Ctx, vei.SubnetID)
		if err != nil {
			return a.Output.AddError(err)
		}

		createSecurityGroupOutput, err := a.CreateSecurityGroup(vei.Ctx, vei.Tags, "osd-network-verifier", vpcId)
		if err != nil {
			return a.Output.AddError(err)
		}

		vei.AWS.SecurityGroupId = *createSecurityGroupOutput.GroupId
		cleanupSecurityGroup = true
	}

	// Create EC2 instance
	instanceID, err := a.createEC2Instance(createEC2InstanceInput{
		amiID:           vei.CloudImageID,
		SubnetID:        vei.SubnetID,
		userdata:        userData,
		KmsKeyID:        vei.AWS.KmsKeyID,
		instanceCount:   instanceCount,
		ctx:             vei.Ctx,
		instanceType:    vei.InstanceType,
		tags:            vei.Tags,
		securityGroupId: vei.AWS.SecurityGroupId,
	})
	if err != nil {
		return a.Output.AddError(err) // fatal
	}

	if err := a.findUnreachableEndpoints(vei.Ctx, instanceID); err != nil {
		a.Output.AddError(err)
	}

	if !vei.SkipInstanceTermination {
		if err := a.AwsClient.TerminateEC2Instance(vei.Ctx, instanceID); err != nil {
			a.Output.AddError(err)
		}
	}

	if cleanupSecurityGroup {
		_, err := a.AwsClient.DeleteSecurityGroup(vei.Ctx, &ec2.DeleteSecurityGroupInput{GroupId: awsTools.String(vei.AWS.SecurityGroupId)})
		if err != nil {
			a.Output.AddError(handledErrors.NewGenericError(err))
			a.Output.AddException(handledErrors.NewGenericError(fmt.Errorf("unable to cleanup security group %s, please manually clean up", vei.AWS.SecurityGroupId)))

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
