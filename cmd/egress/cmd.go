package egress

import (
	"context"
	"fmt"
	"os"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
)

func NewCmdValidateEgress() *cobra.Command {
	config := cloudclient.CmdOptions{}

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
./osd-network-verifier egress --subnet-id $(SUBNET_ID) --profile $(AWS_PROFILE)`,

		Run: func(cmd *cobra.Command, args []string) {
			// ctx
			ctx := context.TODO()

			// Create logger
			builder := ocmlog.NewStdLoggerBuilder()
			builder.Debug(config.Debug)
			logger, err := builder.Build()
			if err != nil {
				fmt.Printf("Unable to build logger: %s\n", err.Error())
				os.Exit(1)
			}
			cli, err := cloudclient.NewClient(ctx, logger, config)
			if err != nil {
				logger.Error(ctx, "Error creating %s cloud client: %s", config.CloudType, err.Error())
				os.Exit(1)
			}

			out := cli.ValidateEgress(ctx)
			out.Summary()
			if !out.IsSuccessful() {
				logger.Error(ctx, "Failure!")
				os.Exit(1)
			}

			logger.Info(ctx, "Success")
		},
	}

	validateEgressCmd.Flags().StringVar(&config.VpcSubnetID, "subnet-id", "", "source subnet ID")
	validateEgressCmd.Flags().StringVar(&config.CloudImageID, "image-id", "", "(optional) cloud image for the compute instance")
	validateEgressCmd.Flags().StringVar(&config.InstanceType, "instance-type", "t3.micro", "(optional) compute instance type")
	validateEgressCmd.Flags().StringVar(&config.Region, "region", config.Region, fmt.Sprintf("(optional) compute instance region. If absent, environment var %[1]v will be used, if set", cloudclient.RegionEnvVarStr, cloudclient.RegionDefault))
	validateEgressCmd.Flags().StringToStringVar(&config.CloudTags, "cloud-tags", cloudclient.DefaultTags, "(optional) comma-seperated list of tags to assign to cloud resources e.g. --cloud-tags key1=value1,key2=value2")
	validateEgressCmd.Flags().BoolVar(&config.Debug, "debug", false, "(optional) if true, enable additional debug-level logging")
	validateEgressCmd.Flags().DurationVar(&config.Timeout, "timeout", cloudclient.DefaultTime, "(optional) timeout for individual egress verification requests")
	validateEgressCmd.Flags().StringVar(&config.KmsKeyID, "kms-key-id", "", "(optional) ID of KMS key used to encrypt root volumes of compute instances. Defaults to cloud account default key")
	validateEgressCmd.Flags().StringVar(&config.AwsProfile, "profile", "", "(optional) AWS profile. If present, any credentials passed with CLI will be ignored.")

	if err := validateEgressCmd.MarkFlagRequired("subnet-id"); err != nil {
		validateEgressCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateEgressCmd
}
