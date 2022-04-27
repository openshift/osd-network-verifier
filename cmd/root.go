package cmd

import (
	"flag"
	"fmt"

	byovpc "github.com/openshift/osd-network-verifier/cmd/byovpc"
	"github.com/openshift/osd-network-verifier/cmd/egress"
	"github.com/spf13/cobra"
)

// GitCommit is the short git commit hash from the environment
var GitCommit string

// Version is the tag version from the environment
var Version string

// NewCmdRoot represents the base command when called without any subcommands
func NewCmdRoot() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:               "osd-network-verifier",
		Example:           "./osd-network-verifier [command] [flags]",
		Version:           fmt.Sprintf("%s, GitCommit: %s", Version, GitCommit),
		Short:             "OSD network verifier CLI",
		Long:              `CLI tool for pre-flight verification of VPC configuration against OSD requirements`,
		DisableAutoGenTag: true,
		Run:               help,
	}

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	// add sub commands
	rootCmd.AddCommand(byovpc.NewCmdByovpc())
	rootCmd.AddCommand(egress.NewCmdValidateEgress())

	return rootCmd
}

func help(cmd *cobra.Command, _ []string) {
	cmd.Help()
}
