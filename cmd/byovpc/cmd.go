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
	cmdOptions := cloudclient.CmdOptions{}

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

			var cli cloudclient.CloudClient
			if cmdOptions.AwsProfile != "" || os.Getenv("AWS_ACCESS_KEY_ID") != "" || cmdOptions.CloudType == "aws" {
				cmdOptions.CloudType = "aws"
				// For AWS type
				if config.awsProfile != "" {
					logger.Info(ctx, "Using AWS profile: %s.", config.awsProfile)
				} else {
					logger.Info(ctx, "Using provided AWS credentials")
				}
				// The use of t3.micro here is arbitrary; we just need to provide any valid machine type
				cli, err = cloudclient.NewClient(ctx, logger, cmdOptions)

			} else {
				//	todo after GCP is implemented, check GCP type using creds
				logger.Error(ctx, "No AWS credentials found.")
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
