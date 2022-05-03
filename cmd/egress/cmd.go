package egress

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
)

var (
	defaultTags            = map[string]string{"osd-network-verifier": "owned", "red-hat-managed": "true", "Name": "osd-network-verifier"}
	regionEnvVarStr string = "AWS_DEFAULT_REGION"
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
		Use: "egress",
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

			logger.Warn(ctx, "Using region: %s", config.region)
			creds := credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))
			cli, err := cloudclient.NewClient(ctx, logger, creds, config.region, config.instanceType, config.cloudTags)
			if err != nil {
				logger.Error(ctx, err.Error())
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

	validateEgressCmd.Flags().StringVar(&config.vpcSubnetID, "subnet-id", "", "ID of the source subnet")
	validateEgressCmd.Flags().StringVar(&config.cloudImageID, "image-id", "", "ID of cloud image")
	validateEgressCmd.Flags().StringVar(&config.instanceType, "instance-type", "t3.micro", "Instance type of the compute instance egress tests are run from")
	validateEgressCmd.Flags().StringVar(&config.region, "region", getDefaultRegion(), fmt.Sprintf("Region to validate. Defaults to exported var %[1]v or '%[2]v' if not %[1]v set", regionEnvVarStr, regionDefault))
	validateEgressCmd.Flags().StringToStringVar(&config.cloudTags, "cloud-tags", defaultTags, "Comma-seperated list of tags to assign to cloud resources")
	validateEgressCmd.Flags().BoolVar(&config.debug, "debug", false, "If true, enable additional debug-level logging")
	validateEgressCmd.Flags().DurationVar(&config.timeout, "timeout", 1*time.Second, "Timeout for individual egress validation requests")
	validateEgressCmd.Flags().StringVar(&config.kmsKeyID, "kms-key-id", "", "ID of KMS key used to encrypt root volumes of created instances. Defaults to cloud account default key")

	if err := validateEgressCmd.MarkFlagRequired("subnet-id"); err != nil {
		validateEgressCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateEgressCmd

}
