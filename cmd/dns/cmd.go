package dns

import (
	"context"
	"fmt"
	"os"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/openshift/osd-network-verifier/pkg/parameters"
	"github.com/spf13/cobra"
)

var vpcId string
var config = cloudclient.CmdOptions{}

func NewCmdValidateDns() *cobra.Command {

	validateDnsCmd := &cobra.Command{
		Use:  "dns",
		RunE: run,
	}

	validateDnsCmd.Flags().StringVar(&vpcId, "vpc-id", "", "ID of the VPC under test")
	validateDnsCmd.Flags().StringVar(&config.Region, "region", cloudclient.RegionDefaultAWS, fmt.Sprintf("Region to validate. Defaults to exported var %[1]v or '%[2]v' if not %[1]v set", cloudclient.RegionEnvVarStrAWS, cloudclient.RegionDefaultAWS))
	validateDnsCmd.Flags().BoolVar(&config.Debug, "debug", false, "If true, enable additional debug-level logging")
	//
	if err := validateDnsCmd.MarkFlagRequired("vpc-id"); err != nil {
		validateDnsCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateDnsCmd

}

func run(cmd *cobra.Command, args []string) error {
	// ctx
	ctx := context.TODO()

	// Create logger
	builder := ocmlog.NewStdLoggerBuilder()
	builder.Debug(config.Debug)
	logger, err := builder.Build()
	if err != nil {
		return fmt.Errorf("Unable to build logger: %s\n", err.Error())
	}

	client, err := cloudclient.GetClientFor(&config)
	if err != nil {
		return fmt.Errorf("Error creating %s cloud client: %s", config.CloudType, err.Error())
	}
	out := client.VerifyDns(parameters.ValidateDns{VpcId: vpcId})
	out.Summary()
	if !out.IsSuccessful() {
		return fmt.Errorf("Failure!")
	}

	logger.Info(ctx, "Success")
	return nil
}
