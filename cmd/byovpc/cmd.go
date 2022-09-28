package byovpc

import (
	"context"
	"fmt"
	"os"

	"github.com/openshift/osd-network-verifier/pkg/verifier"
	awsverifier "github.com/openshift/osd-network-verifier/pkg/verifier/aws"

	"github.com/spf13/cobra"
)

var debug bool

func NewCmdByovpc() *cobra.Command {
	byovpcCmd := &cobra.Command{
		Use:   "byovpc",
		Short: "Verify subnet configuration of a specific VPC",
		Run: func(cmd *cobra.Command, args []string) {

			awsVerifier, err := awsverifier.NewAwsVerifier(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"), os.Getenv("AWS_REGION"), "", debug)
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

	byovpcCmd.Flags().BoolVar(&debug, "debug", false, "If true, enable additional debug-level logging")

	return byovpcCmd
}
