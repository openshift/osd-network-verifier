package byovpc

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/credentials"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
)

var debug bool

func NewCmdByovpc() *cobra.Command {
	byovpcCmd := &cobra.Command{
		Use: "byovpc",
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
			region := os.Getenv("AWS_DEFAULT_REGION")
			tags := map[string]string{}
			cli, err := cloudclient.NewClient(ctx, logger, creds, region, tags)

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
