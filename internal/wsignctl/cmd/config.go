package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/internal/wsignctl/config"
)

// newConfigCmd creates the config command that manages CLI contexts and settings.
// It provides subcommands for viewing and modifying configuration, with a focus
// on managing server contexts for different environments.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long: `The config command provides subcommands for managing wsignctl's
configuration, including contexts for different server endpoints and authentication.

Each context represents a different server environment, allowing you to easily
switch between development, staging, and production servers. Contexts include
server URLs, authentication tokens, and connection settings.`,
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
// It can display either a list of all contexts or detailed information about
// a specific context.
func newConfigGetContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-context [name]",
		Short: "Display one or many contexts",
		Long: `Display information about one or many configuration contexts.

When run without arguments, it displays a table of all available contexts.
When given a context name, it shows detailed information about that specific context.`,
		Example: `  # List all contexts
  wsignctl config get-context

  # Show details for a specific context
  wsignctl config get-context production`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// List all contexts in a formatted table
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

			// Show detailed information for a specific context
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
// It handles all context properties including server URL, authentication,
// and TLS settings.
func newConfigSetContextCmd() *cobra.Command {
	var (
		server            string
		token            string
		insecureSkipTLS  bool
	)

	cmd := &cobra.Command{
		Use:   "set-context NAME",
		Short: "Create or update a context",
		Long: `Create a new context or update an existing one with the specified settings.

A context includes:
- Server URL (required)
- Authentication token (optional)
- TLS verification settings (optional)`,
		Example: `  # Create a new context for development
  wsignctl config set-context dev --server=http://localhost:8080

  # Update production context with authentication
  wsignctl config set-context prod --server=https://signage.example.com --token=mytoken

  # Create context with TLS verification disabled
  wsignctl config set-context staging --server=https://staging.example.com --insecure-skip-tls`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Validate server URL
			if server == "" {
				return fmt.Errorf("server URL is required")
			}

			// Create or update context
			context := &config.Context{
				Name:              name,
				Server:            server,
				Token:            token,
				InsecureSkipVerify: insecureSkipTLS,
			}

			cfg.AddContext(name, context)

			// If this is the first context, automatically make it current
			if cfg.CurrentContext == "" {
				cfg.CurrentContext = name
			}

			// Save configuration
			if err := config.SaveConfig(cfg); err != nil {
				return fmt.Errorf("error saving config: %w", err)
			}

			fmt.Printf("Context %q updated\n", name)
			return nil
		},
	}

	// Add flags for all context properties
	cmd.Flags().StringVar(&server, "server", "", "Server URL (required)")
	cmd.Flags().StringVar(&token, "token", "", "Authentication token")
	cmd.Flags().BoolVar(&insecureSkipTLS, "insecure-skip-tls", false, "Skip TLS certificate verification")

	cmd.MarkFlagRequired("server")

	return cmd
}

// newConfigDeleteContextCmd creates a command for removing contexts from the
// configuration. It includes safety checks to prevent removal of the current context.
func newConfigDeleteContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete-context NAME",
		Short: "Delete a context",
		Long: `Delete a context from the configuration. 
		
If the context is currently active, you must switch to a different context
before deleting it.`,
		Example: `  # Delete the 'staging' context
  wsignctl config delete-context staging`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if err := cfg.RemoveContext(name); err != nil {
				return fmt.Errorf("error removing context: %w", err)
			}

			if err := config.SaveConfig(cfg); err != nil {
				return fmt.Errorf("error saving config: %w", err)
			}

			fmt.Printf("Context %q deleted\n", name)
			return nil
		},
	}
}

// newConfigUseContextCmd creates a command for switching between contexts.
// This allows users to quickly change which server they're interacting with.
func newConfigUseContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use-context NAME",
		Short: "Switch to a different context",
		Long: `Set the current active context for wsignctl.
		
The active context determines which server endpoint will be used for all
commands. This allows quick switching between different environments.`,
		Example: `  # Switch to production context
  wsignctl config use-context production`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if err := cfg.SetCurrentContext(name); err != nil {
				return fmt.Errorf("error setting current context: %w", err)
			}

			if err := config.SaveConfig(cfg); err != nil {
				return fmt.Errorf("error saving config: %w", err)
			}

			fmt.Printf("Switched to context %q\n", name)
			return nil
		},
	}
}

// newConfigViewCmd creates a command for displaying the entire configuration.
// This is useful for debugging and verification purposes.
func newConfigViewCmd() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "view",
		Short: "Display merged configuration",
		Long: `Display the current configuration settings, including all contexts
and which context is active.`,
		Run: func(cmd *cobra.Command, args []string) {
			switch strings.ToLower(outputFormat) {
			case "yaml":
				// TODO: Implement YAML output
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
