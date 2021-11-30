package cmd

import (
	"flag"
	"fmt"

	byovpc "github.com/openshift/osd-network-verifier/cmd/byovpc"
	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// GitCommit is the short git commit hash from the environment
var GitCommit string

// Version is the tag version from the environment
var Version string

// NewCmdRoot represents the base command when called without any subcommands
func NewCmdRoot(streams genericclioptions.IOStreams) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "osd-network-verifier",
		Version:           fmt.Sprintf("%s, GitCommit: %s", Version, GitCommit),
		Short:             "OSD network verifier CLI",
		Long:              `CLI tool to perform some preflight checks for given OSD configurations`,
		DisableAutoGenTag: true,
		Run:               help,
	}

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	// add sub commands
	rootCmd.AddCommand(byovpc.NewCmdByovpc(streams))

	return rootCmd
}

func help(cmd *cobra.Command, _ []string) {
	cmd.Help()
}
