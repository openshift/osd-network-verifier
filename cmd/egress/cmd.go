package egress

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
)

var (
	defaultTags            = map[string]string{"osd-network-verifier": "owned"}
	regionEnvVarStr string = "AWS_DEFAULT_REGION"
	regionDefault   string = "us-east-2"
)

func getDefaultRegion() string {
	val, present := os.LookupEnv(regionEnvVarStr)
	if present {
		return val
	} else {
		return regionDefault
	}
}

func NewCmdValidateEgress() *cobra.Command {
	var vpcSubnetID string
	var cloudImageID string
	var cloudTags map[string]string
	var debug bool
	var region string

	validateEgressCmd := &cobra.Command{
		Use: "egress",
		Run: func(cmd *cobra.Command, args []string) {
			// Create logger
			builder := ocmlog.NewStdLoggerBuilder()
			builder.Debug(debug)
			logger, err := builder.Build()
			if err != nil {
				fmt.Printf("Unable to build logger: %s\n", err.Error())
				os.Exit(1)
			}

			ctx := context.TODO()

			creds := credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))

			logger.Warn(ctx, "Using region: %s", region)
			cli, err := cloudclient.NewClient(ctx, logger, creds, region, cloudTags)
			if err != nil {
				logger.Error(ctx, err.Error())
				os.Exit(1)
			}
			err = cli.ValidateEgress(ctx, vpcSubnetID, cloudImageID)

			if err != nil {
				logger.Error(ctx, err.Error())
				os.Exit(1)
			}

			logger.Info(ctx, "Success")

		},
	}

	validateEgressCmd.Flags().StringVar(&vpcSubnetID, "subnet-id", "", "ID of the source subnet")
	validateEgressCmd.Flags().StringVar(&cloudImageID, "image-id", "", "ID of cloud image")
	validateEgressCmd.Flags().StringVar(&region, "region", getDefaultRegion(), fmt.Sprintf("Region to validate. Defaults to exported var %[1]v or '%[2]v' if not %[1]v set", regionEnvVarStr, regionDefault))
	validateEgressCmd.Flags().StringToStringVar(&cloudTags, "cloud-tags", defaultTags, "Comma-seperated list of tags to assign to cloud resources")
	validateEgressCmd.Flags().BoolVar(&debug, "debug", false, "If true, enable additional debug-level logging")
	validateEgressCmd.MarkFlagRequired("subnet-id")

	return validateEgressCmd

}
