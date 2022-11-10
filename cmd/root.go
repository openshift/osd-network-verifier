package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/osd-network-verifier/cmd/dns"
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
		Use:     "osd-network-verifier",
		Example: "./osd-network-verifier [command] [flags]",
		Version: fmt.Sprintf("%s, GitCommit: %s", Version, GitCommit),
		Short:   "OSD network verifier CLI",
		Long: `CLI tool for pre-flight verification of VPC configuration against OSD requirements. 
For more information see https://github.com/openshift/osd-network-verifier/blob/main/README.md`,
		DisableAutoGenTag: true,
		Run:               help,
	}

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	// add sub commands
	rootCmd.AddCommand(egress.NewCmdValidateEgress())
	rootCmd.AddCommand(dns.NewCmdValidateDns())

	return rootCmd
}

func help(cmd *cobra.Command, _ []string) {
	if err := cmd.Help(); err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
}
