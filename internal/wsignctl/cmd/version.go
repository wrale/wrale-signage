package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// These variables are set during build
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("wsignctl version %s\n", version)
			if debug {
				fmt.Printf("  commit: %s\n", commit)
				fmt.Printf("  built:  %s\n", date)
			}
		},
	}
}
