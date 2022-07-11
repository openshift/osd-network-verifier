package dns

import (
	"context"
	"fmt"
	"os"

	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient/aws"
	"github.com/openshift/osd-network-verifier/pkg/utils"
	"github.com/spf13/cobra"
)

//test params
var vpcId string

//client config
var awsConfig = utils.AWSClientConfig{AwsProfile: "yourProfile"}
var clientConfig = cloudclient.ClientConfig{AWSConfig: &awsConfig}

//execution config
var execConfig = cloudclient.ExecConfig{Debug: true}

func NewCmdValidateDns() *cobra.Command {

	validateDnsCmd := &cobra.Command{
		Use:  "dns",
		RunE: run,
	}

	validateDnsCmd.Flags().StringVar(&vpcId, "vpc-id", "", "ID of the VPC under test")
	validateDnsCmd.Flags().StringVar(&awsConfig.Region, "region", aws.RegionDefaultAWS, fmt.Sprintf("Region to validate. Defaults to exported var %[1]v or '%[2]v' if not %[1]v set", aws.RegionEnvVarStrAWS, aws.RegionDefaultAWS))
	validateDnsCmd.Flags().BoolVar(&execConfig.Debug, "debug", false, "If true, enable additional debug-level logging")
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
	builder.Debug(execConfig.Debug)
	logger, err := builder.Build()
	if err != nil {
		return fmt.Errorf("Unable to build logger: %s\n", err.Error())
	}

	client, err := cloudclient.GetClientFor(&clientConfig, &execConfig)
	if err != nil {
		return fmt.Errorf("Error creating %s cloud client: %s", clientConfig.CloudType, err.Error())
	}
	out := client.VerifyDns(cloudclient.ValidateDns{VpcId: vpcId})
	out.Summary()
	if !out.IsSuccessful() {
		return fmt.Errorf("Failure!")
	}

	logger.Info(ctx, "Success")
	return nil
}
