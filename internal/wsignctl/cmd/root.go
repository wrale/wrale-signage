// Package cmd implements the Wrale Signage CLI commands
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/internal/wsignctl/cmd/content"
	"github.com/wrale/wrale-signage/internal/wsignctl/cmd/display"
	"github.com/wrale/wrale-signage/internal/wsignctl/cmd/rule"
	"github.com/wrale/wrale-signage/internal/wsignctl/config"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "wsignctl",
	Short: "Wrale Signage control tool",
	Long: `wsignctl is a command line tool for managing Wrale Signage displays,
content, and configuration. It provides a complete interface for controlling
your digital signage deployment.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// markFlagRequired is a helper that handles the error from MarkFlagRequired
func markFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Sprintf("Failed to mark flag %q as required: %v", name, err))
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.wsignctl.yaml)")
	rootCmd.PersistentFlags().String("server", "", "API server address")
	rootCmd.PersistentFlags().String("token", "", "Authentication token")
	rootCmd.PersistentFlags().String("context", "", "Configuration context to use")

	// Add commands
	rootCmd.AddCommand(
		display.NewCommand(),
		content.NewCommand(),
		rule.NewCommand(),
		newVersionCmd(),
		newConfigCmd(),
	)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	// Handle command line context override
	if contextName, _ := rootCmd.PersistentFlags().GetString("context"); contextName != "" {
		if err := cfg.SetCurrentContext(contextName); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	// Get current context
	context, err := cfg.GetCurrentContext()
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		return
	}

	// Allow command line flags to override context settings
	if server, _ := rootCmd.PersistentFlags().GetString("server"); server != "" {
		context.Server = server
	}
	if token, _ := rootCmd.PersistentFlags().GetString("token"); token != "" {
		context.Token = token
	}
}
