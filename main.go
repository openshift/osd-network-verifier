package main

import (
	"os"

	"github.com/openshift/osd-network-verifier/cmd"
)

func main() {
	if err := cmd.NewCmdRoot().Execute(); err != nil {
		os.Exit(1)
	}
}
