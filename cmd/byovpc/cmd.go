package byovpc

import (
	"context"
	"os"

	configv1 "github.com/openshift/api/config/v1"
	cloudclient "github.com/openshift/osd-network-verifier/pkg"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdByovpc(streams genericclioptions.IOStreams) *cobra.Command {
	byovpcCmd := &cobra.Command{
		Use: "byovpc",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			// AWS or GCP

			caller := configv1.AWSPlatformType // testing only, assuming it's aws, check what was provided actually.
			var cli cloudclient.CloudClient

			switch {
			case caller == configv1.AWSPlatformType:
				cli = cloudclient.GetClientFor(configv1.AWSPlatformType)
			case caller == configv1.GCPPlatformType:
				cli = cloudclient.GetClientFor(configv1.GCPPlatformType)
			}

			err := cli.ByoVPCValidator(context.TODO())
			if err != nil {
				streams.ErrOut.Write([]byte(err.Error()))
				os.Exit(1)
			}

			streams.Out.Write([]byte("success"))

		},
	}

	return byovpcCmd
}
