package dns

import (
	"context"
	"fmt"
	"os"

	"github.com/openshift/osd-network-verifier/cmd/utils"
	"github.com/openshift/osd-network-verifier/pkg/verifier"
	"github.com/spf13/cobra"
)

var (
	regionEnvVarStr string = "AWS_DEFAULT_REGION"
	regionDefault   string = "us-east-2"
)

type dnsConfig struct {
	vpcID      string
	debug      bool
	region     string
	awsProfile string
}

func getDefaultRegion() string {
	val, present := os.LookupEnv(regionEnvVarStr)
	if present {
		return val
	} else {
		return regionDefault
	}
}

func NewCmdValidateDns() *cobra.Command {
	config := dnsConfig{}

	validateDnsCmd := &cobra.Command{
		Use: "dns",
		Run: func(cmd *cobra.Command, args []string) {

			awsVerifier, err := utils.GetAwsVerifier(os.Getenv("AWS_REGION"), config.awsProfile, config.debug)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			awsVerifier.Logger.Warn(context.TODO(), "Using region: %s", config.region)

			vdi := verifier.VerifyDnsInput{
				VpcID: config.vpcID,
				Ctx:   context.TODO(),
			}
			out := verifier.VerifyDns(awsVerifier, vdi)
			out.Summary(config.debug)
			if !out.IsSuccessful() {
				awsVerifier.Logger.Error(context.TODO(), "Failure!")
				os.Exit(1)
			}

			awsVerifier.Logger.Info(context.TODO(), "Success")
		},
	}

	validateDnsCmd.Flags().StringVar(&config.vpcID, "vpc-id", "", "ID of the VPC under test")
	validateDnsCmd.Flags().StringVar(&config.region, "region", getDefaultRegion(), fmt.Sprintf("Region to validate. Defaults to exported var %[1]v or '%[2]v' if not %[1]v set", regionEnvVarStr, regionDefault))
	validateDnsCmd.Flags().BoolVar(&config.debug, "debug", false, "If true, enable additional debug-level logging")
	validateDnsCmd.Flags().StringVar(&config.awsProfile, "profile", "", "(optional) AWS profile. If present, any credentials passed with CLI will be ignored.")

	if err := validateDnsCmd.MarkFlagRequired("vpc-id"); err != nil {
		validateDnsCmd.PrintErr(err)
		os.Exit(1)
	}

	return validateDnsCmd

}
