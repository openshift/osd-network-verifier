package egress

import (
	"context"
	"fmt"
	"os"
	"time"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
)

var (
	defaultTags            = map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "Name": "osd-network-verifier"}
	regionEnvVarStr string = "AWS_REGION"
	regionDefault   string = "us-east-2"
)

type egressConfig struct {
	vpcSubnetID  string
	cloudImageID string
	instanceType string
	cloudTags    map[string]string
	debug        bool
	region       string
	timeout      time.Duration
	kmsKeyID     string
	awsProfile   string
}

func getDefaultRegion() string {
	val, present := os.LookupEnv(regionEnvVarStr)
	if present {
		return val
	} else {
		return regionDefault
	}
}
func NewCmdValidateEgress() *cobra.Command {
	config := egressConfig{}

	validateEgressCmd := &cobra.Command{
		Use:   "egress",
		Short: "Verify essential openshift domains are reachable from given subnet ID.",
		Long:  `Verify essential openshift domains are reachable from given subnet ID.`,
		Example: `For AWS, ensure your credential environment vars 
AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY (also AWS_SESSION_TOKEN for STS credentials) 
are set correctly before execution.

# Verify that essential openshift domains are reachable from a given SUBNET_ID
./osd-network-verifier egress --subnet-id $(SUBNET_ID) --image-id $(IMAGE_ID)`,
		Run: func(cmd *cobra.Command, args []string) {
			// ctx
			ctx := context.TODO()

			// Create logger
			builder := ocmlog.NewStdLoggerBuilder()
			builder.Debug(config.debug)
			logger, err := builder.Build()
			if err != nil {
				fmt.Printf("Unable to build logger: %s\n", err.Error())
				os.Exit(1)
			}
			logger.Info(ctx, "Using region: %s", config.region)
			cli, err := NewClient(ctx, logger, config)
			if err != nil {
				logger.Error(ctx, "Error creating cloud client: %s", err.Error())
				os.Exit(1)
			}
			out := cli.ValidateEgress(ctx, config.vpcSubnetID, config.cloudImageID, config.kmsKeyID, config.timeout)
			out.Summary()
			if !out.IsSuccessful() {
				logger.Error(ctx, "Failure!")
				os.Exit(1)
			}

			logger.Info(ctx, "Success")
		},
	}

	validateEgressCmd.Flags().StringVar(&config.vpcSubnetID, "subnet-id", "", "source subnet ID")
	validateEgressCmd.Flags().StringVar(&config.cloudImageID, "image-id", "", "(optional) cloud image for the compute instance")
	validateEgressCmd.Flags().StringVar(&config.instanceType, "instance-type", "t3.micro", "(optional) compute instance type")
	validateEgressCmd.Flags().StringVar(&config.region, "region", getDefaultRegion(), fmt.Sprintf("(optional) compute instance region. If absent, environment var %[1]v will be used, if set", regionEnvVarStr, regionDefault))
	validateEgressCmd.Flags().StringToStringVar(&config.cloudTags, "cloud-tags", defaultTags, "(optional) comma-seperated list of tags to assign to cloud resources e.g. --cloud-tags key1=value1,key2=value2")
	validateEgressCmd.Flags().BoolVar(&config.debug, "debug", false, "(optional) if true, enable additional debug-level logging")
	validateEgressCmd.Flags().DurationVar(&config.timeout, "timeout", 1*time.Second, "(optional) timeout for individual egress verification requests")
	validateEgressCmd.Flags().StringVar(&config.kmsKeyID, "kms-key-id", "", "(optional) ID of KMS key used to encrypt root volumes of compute instances. Defaults to cloud account default key")
	validateEgressCmd.Flags().StringVar(&config.awsProfile, "profile", "", "(optional) AWS profile. If present, any credentials passed with CLI will be ignored.")

	if err := validateEgressCmd.MarkFlagRequired("subnet-id"); err != nil {
		validateEgressCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateEgressCmd

}

func NewClient(ctx context.Context, logger ocmlog.Logger, config egressConfig) (cloudclient.CloudClient, error) {
	if config.awsProfile != "" || os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		// For AWS type
		if config.awsProfile != "" {
			logger.Info(ctx, "Using AWS profile: %s.", config.awsProfile)
		} else {
			logger.Info(ctx, "Using provided AWS credentials")
		}
		return cloudclient.NewClient(ctx, logger, config.region, config.instanceType, config.cloudTags, "aws",
			config.awsProfile)

	} else {
		//	todo after GCP is implemented, check GCP type using creds
		return nil, fmt.Errorf("No AWS credentials found.")
	}
}
