package byovpc

import (
	_ "context"
	_ "fmt"
	"github.com/spf13/cobra"
	_ "os"
)

var debug bool

func NewCmdByovpc() *cobra.Command {

	byovpcCmd := &cobra.Command{
		Use:   "byovpc",
		Short: "Verify subnet configuration of a specific VPC",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	byovpcCmd.Flags().BoolVar(&debug, "debug", false, "If true, enable additional debug-level logging")

	return byovpcCmd
}
