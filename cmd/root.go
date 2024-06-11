package cmd

import (
	"flag"
	"fmt"
	"github.com/openshift/osd-network-verifier/cmd/dns"
	"github.com/openshift/osd-network-verifier/cmd/egress"
	"github.com/openshift/osd-network-verifier/version"
	"github.com/spf13/cobra"
	"os"
)

type verifierOptions struct {
	debug bool
}

// NewCmdRoot represents the base command when called without any subcommands
func NewCmdRoot() *cobra.Command {
	opts := verifierOptions{}
	rootCmd := &cobra.Command{
		Use:     "osd-network-verifier",
		Example: "./osd-network-verifier [command] [flags]",
		Short:   "OSD network verifier CLI",
		Version: fmt.Sprintf("%v@%v", version.Version, version.ShortCommitHash),
		Long: `CLI tool for pre-flight verification of VPC configuration against OSD requirements. 
For more information see https://github.com/openshift/osd-network-verifier/blob/main/README.md`,
		DisableAutoGenTag: true,
		PersistentPreRun: func(*cobra.Command, []string) {
			if opts.debug {
				fmt.Printf("Version:\t%v\nCommit Hash:\t%v\n", version.Version, version.CommitHash)
			}
		},
		Run: help,
	}

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	rootCmd.PersistentFlags().BoolVar(&opts.debug, "debug", false, "")

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
