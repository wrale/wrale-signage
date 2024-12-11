package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is set during build
	Version = "dev"
	// Commit is set during build
	Commit = "none"
	// BuildDate is set during build
	BuildDate = "unknown"

	debugVersion bool
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			if debugVersion {
				fmt.Printf("Version:\t%s\nCommit:\t\t%s\nBuild Date:\t%s\n",
					Version, Commit, BuildDate)
			} else {
				fmt.Printf("wsignctl version %s\n", Version)
			}
		},
	}

	cmd.Flags().BoolVar(&debugVersion, "debug", false, "Show detailed version information")
	return cmd
}
