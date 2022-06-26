package dns

import (
	"context"
	"fmt"
	"os"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
)

var (
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
func NewCmdValidateDns() *cobra.Command {
	config := cloudclient.CmdOptions{}

	validateDnsCmd := &cobra.Command{
		Use: "dns",
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
			out := cli.VerifyDns(ctx, config.VpcSubnetID)
			out.Summary()
			if !out.IsSuccessful() {
				logger.Error(ctx, "Failure!")
				os.Exit(1)
			}

			logger.Info(ctx, "Success")
		},
	}

	validateDnsCmd.Flags().StringVar(&config.VpcSubnetID, "vpc-id", "", "ID of the VPC under test")
	validateDnsCmd.Flags().StringVar(&config.Region, "region", getDefaultRegion(), fmt.Sprintf("Region to validate. Defaults to exported var %[1]v or '%[2]v' if not %[1]v set", regionEnvVarStr, regionDefault))
	validateDnsCmd.Flags().BoolVar(&config.Debug, "debug", false, "If true, enable additional debug-level logging")

	if err := validateDnsCmd.MarkFlagRequired("vpc-id"); err != nil {
		validateDnsCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateDnsCmd

}
