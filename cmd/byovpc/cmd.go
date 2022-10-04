package byovpc

import (
	"context"
	"fmt"
	"os"

	"github.com/openshift/osd-network-verifier/cmd/utils"
	"github.com/openshift/osd-network-verifier/pkg/verifier"

	"github.com/spf13/cobra"
)

var debug bool
var awsProfile string

func NewCmdByovpc() *cobra.Command {
	byovpcCmd := &cobra.Command{
		Use:   "byovpc",
		Short: "Verify subnet configuration of a specific VPC",
		Run: func(cmd *cobra.Command, args []string) {

			awsVerifier, err := utils.GetAwsVerifier(os.Getenv("AWS_REGION"), awsProfile, debug)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			awsVerifier.Logger.Warn(context.TODO(), "Using region: %s", os.Getenv("AWS_REGION"))

			err = verifier.ByoVPCValidator(awsVerifier, verifier.ByoVPCValidatorInput{Ctx: context.TODO()})
			if err != nil {
				awsVerifier.Logger.Error(context.TODO(), err.Error())
				os.Exit(1)
			}

			awsVerifier.Logger.Info(context.TODO(), "Success")
		},
	}

	byovpcCmd.Flags().StringVar(&awsProfile, "profile", "", "(optional) AWS profile. If present, any credentials passed with CLI will be ignored.")
	byovpcCmd.Flags().BoolVar(&debug, "debug", false, "If true, enable additional debug-level logging")

	return byovpcCmd
}
