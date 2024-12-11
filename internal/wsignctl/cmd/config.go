package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/internal/wsignctl/config"
)

// newConfigCmd creates the config command that manages CLI contexts and settings.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long: `The config command provides subcommands for managing wsignctl's
configuration, including contexts for different server endpoints and authentication.`,
	}

	cmd.AddCommand(
		newConfigGetContextCmd(),
		newConfigSetContextCmd(),
		newConfigDeleteContextCmd(),
		newConfigUseContextCmd(),
		newConfigViewCmd(),
	)

	return cmd
}

// newConfigGetContextCmd creates a command for viewing context information.
func newConfigGetContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-context [name]",
		Short: "Display one or many contexts",
		Long:  `Display information about one or many configuration contexts.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Printf("CURRENT   NAME           SERVER\n")
				for name, ctx := range cfg.Contexts {
					current := " "
					if name == cfg.CurrentContext {
						current = "*"
					}
					fmt.Printf("%-8s  %-13s  %s\n", current, name, ctx.Server)
				}
				return
			}

			name := args[0]
			ctx, ok := cfg.Contexts[name]
			if !ok {
				fmt.Printf("Error: context %q not found\n", name)
				return
			}

			fmt.Printf("Name: %s\n", name)
			fmt.Printf("Server: %s\n", ctx.Server)
			fmt.Printf("Insecure Skip Verify: %v\n", ctx.InsecureSkipVerify)
			if ctx.Token != "" {
				fmt.Printf("Token: %s...\n", ctx.Token[:10])
			}
		},
	}
}

// newConfigSetContextCmd creates a command for creating or updating contexts.
func newConfigSetContextCmd() *cobra.Command {
	var (
		server          string
		token           string
		insecureSkipTLS bool
	)

	cmd := &cobra.Command{
		Use:   "set-context NAME",
		Short: "Create or update a context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if server == "" {
				return fmt.Errorf("server URL is required")
			}

			context := &config.Context{
				Name:               name,
				Server:             server,
				Token:              token,
				InsecureSkipVerify: insecureSkipTLS,
			}

			cfg.AddContext(name, context)

			if cfg.CurrentContext == "" {
				cfg.CurrentContext = name
			}

			if err := config.SaveConfig(cfg, cfgFile); err != nil {
				return fmt.Errorf("error saving config: %w", err)
			}

			fmt.Printf("Context %q updated\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Server URL (required)")
	cmd.Flags().StringVar(&token, "token", "", "Authentication token")
	cmd.Flags().BoolVar(&insecureSkipTLS, "insecure-skip-tls", false, "Skip TLS certificate verification")

	markFlagRequired(cmd, "server")

	return cmd
}

// newConfigDeleteContextCmd creates a command for removing contexts.
func newConfigDeleteContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-context NAME",
		Short: "Delete a context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if err := cfg.RemoveContext(name); err != nil {
				return fmt.Errorf("error removing context: %w", err)
			}

			if err := config.SaveConfig(cfg, cfgFile); err != nil {
				return fmt.Errorf("error saving config: %w", err)
			}

			fmt.Printf("Context %q deleted\n", name)
			return nil
		},
	}
}

// newConfigUseContextCmd creates a command for switching between contexts.
func newConfigUseContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use-context NAME",
		Short: "Switch to a different context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if err := cfg.SetCurrentContext(name); err != nil {
				return fmt.Errorf("error setting current context: %w", err)
			}

			if err := config.SaveConfig(cfg, cfgFile); err != nil {
				return fmt.Errorf("error saving config: %w", err)
			}

			fmt.Printf("Switched to context %q\n", name)
			return nil
		},
	}
}

// newConfigViewCmd creates a command for displaying the configuration.
func newConfigViewCmd() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "view",
		Short: "Display merged configuration",
		Run: func(cmd *cobra.Command, args []string) {
			switch strings.ToLower(outputFormat) {
			case "yaml":
				fmt.Println("YAML output not yet implemented")
			default:
				fmt.Printf("Current Context: %s\n\n", cfg.CurrentContext)
				fmt.Printf("Contexts:\n")
				for name, ctx := range cfg.Contexts {
					fmt.Printf("- %s:\n", name)
					fmt.Printf("    Server: %s\n", ctx.Server)
					fmt.Printf("    InsecureSkipVerify: %v\n", ctx.InsecureSkipVerify)
					if ctx.Token != "" {
						fmt.Printf("    Token: %s...\n", ctx.Token[:10])
					}
				}
			}
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, yaml)")

	return cmd
}
