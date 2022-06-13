package byovpc

import (
	"context"
	"fmt"
	"os"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
)

var debug bool

type byovpcConfig struct {
	awsProfile string
}

func NewCmdByovpc() *cobra.Command {
	config := byovpcConfig{}
	byovpcCmd := &cobra.Command{
		Use:   "byovpc",
		Short: "Verify subnet configuration of a specific VPC",
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

			// TODO when this command is actually used, most if not all of the following should be command line options
			region := os.Getenv("AWS_REGION")
			instanceType := "t3.micro"
			tags := map[string]string{}

			var cli cloudclient.CloudClient
			if config.awsProfile != "" || os.Getenv("AWS_ACCESS_KEY_ID") != "" {
				// For AWS type
				if config.awsProfile != "" {
					logger.Info(ctx, "Using AWS profile: %s.", config.awsProfile)
				} else {
					logger.Info(ctx, "Using provided AWS credentials")
				}
				// The use of t3.micro here is arbitrary; we just need to provide any valid machine type
				cli, err = cloudclient.NewClient(ctx, logger, region, instanceType, tags, "aws", config.awsProfile)

			} else {
				//	todo after GCP is implemented, check GCP type using creds
				logger.Info(ctx, "GCP cloud credentials found.")
				cli, err = cloudclient.NewClient(ctx, logger, region, "", nil, "gcp", config.awsProfile)
			}
			if err != nil {
				logger.Error(ctx, err.Error())
				os.Exit(1)
			}

			err = cli.ByoVPCValidator(ctx)
			if err != nil {
				logger.Error(ctx, err.Error())
				os.Exit(1)
			}

			logger.Info(ctx, "Success")
		},
	}

	byovpcCmd.Flags().BoolVar(&debug, "debug", false, "If true, enable additional debug-level logging")

	return byovpcCmd
}
