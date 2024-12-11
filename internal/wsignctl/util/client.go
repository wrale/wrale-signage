// Package util provides utility functions for the wsignctl command line tool.
package util

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/internal/wsignctl/client"
	"github.com/wrale/wrale-signage/internal/wsignctl/config"
)

// clientConfig holds the configuration needed to create an API client
type clientConfig struct {
	apiURL string
	token  string
}

// GetClient creates a new API client configured from the environment and config file.
// This function does not use command-line flags.
func GetClient() (*client.Client, error) {
	cfg, err := getClientConfig(nil)
	if err != nil {
		return nil, err
	}

	return createClient(cfg)
}

// GetClientFromCommand creates a new API client using configuration from command flags,
// environment variables, and config file (in that order of precedence).
func GetClientFromCommand(cmd *cobra.Command) (*client.Client, error) {
	cfg, err := getClientConfig(cmd)
	if err != nil {
		return nil, err
	}

	return createClient(cfg)
}

// getClientConfig retrieves client configuration from available sources in order of precedence:
// 1. Command flags (if cmd is provided)
// 2. Environment variables
// 3. Configuration file
func getClientConfig(cmd *cobra.Command) (*clientConfig, error) {
	cfg := &clientConfig{}

	// Try command flags first if available
	var configPath string
	if cmd != nil {
		if server, err := cmd.Flags().GetString("server"); err == nil && server != "" {
			cfg.apiURL = server
		}
		if token, err := cmd.Flags().GetString("token"); err == nil && token != "" {
			cfg.token = token
		}
		if path, err := cmd.Flags().GetString("config"); err == nil {
			configPath = path
		}
	}

	// Check environment variables next
	if cfg.apiURL == "" {
		cfg.apiURL = os.Getenv("WRALE_API_URL")
	}
	if cfg.token == "" {
		cfg.token = os.Getenv("WRALE_AUTH_TOKEN")
	}

	// If still missing values, try config file
	if cfg.apiURL == "" || cfg.token == "" {
		fileCfg, err := config.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}

		ctx, err := fileCfg.GetCurrentContext()
		if err != nil {
			return nil, fmt.Errorf("failed to get current context: %w", err)
		}

		// Use context values if still not set
		if cfg.apiURL == "" {
			if ctx.Server == "" {
				return nil, fmt.Errorf("no API server configured - set WRALE_API_URL, use --server flag, or configure server in wsignctl config")
			}
			cfg.apiURL = ctx.Server
		}

		if cfg.token == "" {
			if ctx.Token == "" {
				return nil, fmt.Errorf("no auth token configured - set WRALE_AUTH_TOKEN, use --token flag, or authenticate using 'wsignctl login'")
			}
			cfg.token = ctx.Token
		}
	}

	return cfg, nil
}

// createClient creates a new API client using the provided configuration
func createClient(cfg *clientConfig) (*client.Client, error) {
	c, err := client.NewClient(
		cfg.apiURL,
		client.WithToken(cfg.token),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return c, nil
}