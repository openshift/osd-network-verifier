package egress

import (
	"context"
	"fmt"
	"os"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/openshift/osd-network-verifier/pkg/parameters"
	"github.com/spf13/cobra"
)

var config = cloudclient.CmdOptions{}
var inputStruct = egressInput{}

type egressInput struct {
	vpcSubnetId string
}

func NewCmdValidateEgress() *cobra.Command {
	validateEgressCmd := &cobra.Command{
		Use:        "egress",
		Aliases:    nil,
		SuggestFor: nil,
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
	validateEgressCmd.Flags().StringVar(&config.CloudImageID, "image-id", "", "(optional) cloud image for the compute instance")
	validateEgressCmd.Flags().StringVar(&config.InstanceType, "instance-type", "t3.micro", "(optional) compute instance type")
	validateEgressCmd.Flags().StringVar(&config.Region, "region", config.Region, fmt.Sprintf("(optional) compute instance region. If absent, environment var %[1]v will be used, if set", cloudclient.RegionEnvVarStrAWS, cloudclient.RegionDefaultAWS))
	validateEgressCmd.Flags().StringToStringVar(&config.CloudTags, "cloud-tags", cloudclient.DefaultTagsAWS, "(optional) comma-seperated list of tags to assign to cloud resources e.g. --cloud-tags key1=value1,key2=value2")
	validateEgressCmd.Flags().BoolVar(&config.Debug, "debug", false, "(optional) if true, enable additional debug-level logging")
	validateEgressCmd.Flags().DurationVar(&config.Timeout, "timeout", cloudclient.DefaultTime, "(optional) timeout for individual egress verification requests")
	validateEgressCmd.Flags().StringVar(&config.KmsKeyID, "kms-key-id", "", "(optional) ID of KMS key used to encrypt root volumes of compute instances. Defaults to cloud account default key")

	if err := validateEgressCmd.MarkFlagRequired("subnet-id"); err != nil {
		validateEgressCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateEgressCmd
}

func rune(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create logger
	logger, err := ocmlog.NewStdLoggerBuilder().Debug(config.Debug).Build()
	if err != nil {
		return fmt.Errorf("unable to build logger: %s\n", err.Error())
	}
	config.Logger = logger
	config.Ctx = ctx

	client, err := cloudclient.GetClientFor(&config)
	if err != nil {
		return fmt.Errorf("error creating cloud client: %s", err.Error())
	}

	//Downstream must pass in data in the form defined in parameters.ValidateEgress
	out := client.ValidateEgress(parameters.ValidateEgress{
		VpcSubnetID: inputStruct.vpcSubnetId, // required downstream argument
	})

	out.Summary()
	if !out.IsSuccessful() {
		return fmt.Errorf("Failure!")
	}

	logger.Info(ctx, "Success")
	return nil
}
