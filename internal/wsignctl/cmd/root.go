// Package cmd implements the Wrale Signage CLI commands
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/internal/wsignctl/config"
)

var (
	// Global configuration
	cfg *config.Config

	// Global flags
	cfgFile string
	debug   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wsignctl",
	Short: "Wrale Signage CLI",
	Long: `wsignctl is a command line interface for managing Wrale Signage displays
and content. It provides commands for registering displays, managing content,
and controlling the signage system.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		var err error
		if cfgFile != "" {
			os.Setenv("WSIGNCTL_CONFIG", cfgFile)
		}
		cfg, err = config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.wsignctl/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")

	// Add commands
	rootCmd.AddCommand(
		newConfigCmd(),
		newDisplayCmd(),
		newVersionCmd(),
	)
}
