package byovpc

import (
	// configv1 "github.com/openshift/api/config/v1"
	// "github.com/openshift/osd-network-verifier/pkg/cloudclient"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func NewCmdByovpc(streams genericclioptions.IOStreams) *cobra.Command {
	byovpcCmd := &cobra.Command{
		Use: "byovpc",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			// AWS & GCP
			// cloudclient.GetClientFor(configv1.AWSPlatformType)
			// cloudclient.ByoVPCValidator()

			streams.Out.Write([]byte("success"))

		},
	}

	return byovpcCmd
}
