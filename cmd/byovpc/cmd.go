package byovpc

import (

	// "github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdByovpc(streams genericclioptions.IOStreams) *cobra.Command {
	byovpcCmd := &cobra.Command{
		Use: "byovpc",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			// AWS or GCP

			// switch {
			// case caller == configv1.AWSPlatformType:
			// 	cloudclient.GetClientFor(configv1.AWSPlatformType)
			// case caller == configv1.GCPPlatformType:
			// 	cloudclient.GetClientFor(configv1.GCPPlatformType)
			// }

			// cloudclient.ByoVPCValidator()

			streams.Out.Write([]byte("success"))

		},
	}

	return byovpcCmd
}
