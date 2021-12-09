package egress

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdValidateEgress(streams genericclioptions.IOStreams) *cobra.Command {
	var vpcSubnetID string
	var cloudImageID string

	validateEgressCmd := &cobra.Command{
		Use: "egress",
		Run: func(cmd *cobra.Command, args []string) {
			creds := credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))
			region := os.Getenv("AWS_DEFAULT_REGION")
			cli, err := cloudclient.NewClient(creds, region)

			if err != nil {
				streams.ErrOut.Write([]byte(err.Error()))
				os.Exit(1)
			}
			err = cli.ValidateEgress(context.TODO(), vpcSubnetID, cloudImageID)

			if err != nil {
				streams.ErrOut.Write([]byte(err.Error()))
				os.Exit(1)
			}

			streams.Out.Write([]byte("success"))

		},
	}

	validateEgressCmd.Flags().StringVar(&vpcSubnetID, "subnet-id", "", "ID of the source subnet")
	validateEgressCmd.Flags().StringVar(&cloudImageID, "image-id", "", "ID of cloud image")
	validateEgressCmd.MarkFlagRequired("subnet-id")

	return validateEgressCmd

}
