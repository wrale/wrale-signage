// Package cmd implements the Wrale Signage CLI commands
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

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

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.wsignctl.yaml)")
	rootCmd.PersistentFlags().String("server", "", "API server address")
	rootCmd.PersistentFlags().String("token", "", "Authentication token")

	// Add commands
	rootCmd.AddCommand(newDisplayCmd())
	rootCmd.AddCommand(newContentCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newRuleCmd())
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	// Allow command line flags to override config file
	if server, _ := rootCmd.PersistentFlags().GetString("server"); server != "" {
		cfg.Server = server
	}
	if token, _ := rootCmd.PersistentFlags().GetString("token"); token != "" {
		cfg.Token = token
	}
}
