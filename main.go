package main

import (
	"flag"
	"os"

	"github.com/openshift/osd-network-verifier/cmd"
	"github.com/spf13/pflag"
)

func main() {

	flags := pflag.NewFlagSet("osd-network-verifier", pflag.ExitOnError)
	flag.CommandLine.Parse([]string{})
	pflag.CommandLine = flags

	if err := cmd.NewCmdRoot().Execute(); err != nil {
		os.Exit(1)
	}
}
