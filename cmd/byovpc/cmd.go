package byovpc

import (
	"github.com/spf13/cobra"
)

var debug bool

func NewCmdByovpc() *cobra.Command {

	byovpcCmd := &cobra.Command{
		Use:   "byovpc",
		Short: "Verify subnet configuration of a specific VPC",
		Run:   func(cmd *cobra.Command, args []string) { return },
	}

	byovpcCmd.Flags().BoolVar(&debug, "debug", false, "If true, enable additional debug-level logging")

	return byovpcCmd
}
