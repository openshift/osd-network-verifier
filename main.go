package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openshift/osd-network-verifier/cmd"
	"github.com/spf13/pflag"
)

func main() {

	flags := pflag.NewFlagSet("osd-network-verifier", pflag.ExitOnError)
	flag.CommandLine.Parse([]string{})
	pflag.CommandLine = flags

	//command := cmd.NewCmdRoot(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	command := cmd.NewCmdRoot()

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
