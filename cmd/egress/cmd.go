package egress

import (
	"context"
	"os"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdValidateEgress(streams genericclioptions.IOStreams) *cobra.Command {
	validateEgressCmd := &cobra.Command{
		Use: "egress",
		Run: func(cmd *cobra.Command, args []string) {

			caller := configv1.AWSPlatformType // testing only, assuming it's aws, check what was provided actually.
			var cli cloudclient.CloudClient

			switch {
			case caller == configv1.AWSPlatformType:
				cli = cloudclient.GetClientFor(configv1.AWSPlatformType)
			case caller == configv1.GCPPlatformType:
				cli = cloudclient.GetClientFor(configv1.GCPPlatformType)
			}
			subnetid, _ := cmd.Flags().GetString("subnet-id")

			err := cli.ValidateEgress(context.TODO(), subnetid)

			if err != nil {
				streams.ErrOut.Write([]byte(err.Error()))
				os.Exit(1)
			}

			streams.Out.Write([]byte("success"))

		},
	}

	validateEgressCmd.Flags().StringP("subnet-id", "", "", "ID of the source subnet")
	validateEgressCmd.MarkFlagRequired("subnet-id")

	return validateEgressCmd

}
