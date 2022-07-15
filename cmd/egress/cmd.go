package egress

import (
	"context"
	"fmt"
	"os"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	awsCloudClient "github.com/openshift/osd-network-verifier/pkg/cloudclient/aws"
	"github.com/openshift/osd-network-verifier/pkg/utils"
	"github.com/spf13/cobra"
)

// test params
type egressInput struct {
	vpcSubnetId string
}

var inputStruct = egressInput{}

//client config
var awsConfig = utils.AWSClientConfig{}
var clientConfig = cloudclient.ClientConfig{AWSConfig: &awsConfig}

//execution config
var execConfig = cloudclient.ExecConfig{Debug: true}

func NewCmdValidateEgress() *cobra.Command {
	validateEgressCmd := &cobra.Command{
		Use:        "egress",
		Short:      "Verify essential openshift domains are reachable from given subnet ID.",
		Long:       `Verify essential openshift domains are reachable from given subnet ID.`,
		Example: `For AWS, ensure your credential environment vars 
AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY (also AWS_SESSION_TOKEN for STS credentials) 
are set correctly before execution.

# Verify that essential openshift domains are reachable from a given SUBNET_ID
./osd-network-verifier egress --subnet-id $(SUBNET_ID) --image-id $(IMAGE_ID)`,
		RunE: rune,
	}

	//test specific args - required
	validateEgressCmd.Flags().StringVar(&inputStruct.vpcSubnetId, "subnet-id", "", "source subnet ID")

	//client args - all these have defaults
	validateEgressCmd.Flags().StringVar(&awsConfig.CloudImageID, "image-id", "", "(optional) cloud image for the compute instance")
	validateEgressCmd.Flags().StringVar(&awsConfig.InstanceType, "instance-type", "t3.micro", "(optional) compute instance type")
	validateEgressCmd.Flags().StringVar(&awsConfig.Region, "region", awsConfig.Region, fmt.Sprintf("(optional) compute instance region. If absent, environment var %[1]v will be used, if set", awsCloudClient.RegionEnvVarStrAWS, awsCloudClient.RegionDefaultAWS))
	validateEgressCmd.Flags().StringToStringVar(&awsConfig.CloudTags, "cloud-tags", awsCloudClient.DefaultTagsAWS, "(optional) comma-seperated list of tags to assign to cloud resources e.g. --cloud-tags key1=value1,key2=value2")
	validateEgressCmd.Flags().BoolVar(&execConfig.Debug, "debug", false, "(optional) if true, enable additional debug-level logging")
	validateEgressCmd.Flags().DurationVar(&execConfig.Timeout, "timeout", cloudclient.DefaultTime, "(optional) timeout for individual egress verification requests")
	validateEgressCmd.Flags().StringVar(&awsConfig.KmsKeyID, "kms-key-id", "", "(optional) ID of KMS key used to encrypt root volumes of compute instances. Defaults to cloud account default key")
	validateEgressCmd.Flags().StringVar(&awsConfig.AwsProfile, "profile", "", "(optional) AWS profile. If present, any credentials passed with CLI will be ignored.")

	if err := validateEgressCmd.MarkFlagRequired("subnet-id"); err != nil {
		validateEgressCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateEgressCmd
}

func rune(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create logger
	logger, err := ocmlog.NewStdLoggerBuilder().Debug(execConfig.Debug).Build()
	if err != nil {
		return fmt.Errorf("unable to build logger: %s\n", err.Error())
	}
	execConfig.Logger = logger
	execConfig.Ctx = ctx

	client, err := cloudclient.GetClientFor(&clientConfig, &execConfig)
	if err != nil {
		return fmt.Errorf("error creating cloud client: %s", err.Error())
	}

	//Downstream must pass in data in the form defined in parameters.ValidateEgress
	out := client.ValidateEgress(cloudclient.ValidateEgress{
		VpcSubnetID: inputStruct.vpcSubnetId, // required downstream argument
	})

	out.Summary()
	if !out.IsSuccessful() {
		return fmt.Errorf("Failure!")
	}

	logger.Info(ctx, "Success")
	return nil
}
