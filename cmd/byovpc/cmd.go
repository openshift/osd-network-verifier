package byovpc

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdByovpc(streams genericclioptions.IOStreams) *cobra.Command {
	byovpcCmd := &cobra.Command{
		Use: "byovpc",
		Run: func(cmd *cobra.Command, args []string) {
			creds := credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))
			region := os.Getenv("AWS_DEFAULT_REGION")
			cli, err := cloudclient.NewClient(creds, region)

			err = cli.ByoVPCValidator(context.TODO())
			if err != nil {
				streams.ErrOut.Write([]byte(err.Error()))
				os.Exit(1)
			}

			streams.Out.Write([]byte("success"))

		},
	}

	return byovpcCmd
}
