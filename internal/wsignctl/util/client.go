package util

import (
	"fmt"
	"os"

	"github.com/wrale/wrale-signage/internal/wsignctl/client"
	"github.com/wrale/wrale-signage/internal/wsignctl/config"
)

// GetClient creates a new API client configured from the environment and config
func GetClient() (*client.Client, error) {
	// Check for API URL override in environment
	apiURL := os.Getenv("WRALE_API_URL")
	if apiURL == "" {
		// Load from config if not in environment
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		if cfg.Server == "" {
			return nil, fmt.Errorf("no API server configured - set WRALE_API_URL or configure in wsignctl config")
		}
		apiURL = cfg.Server
	}

	// Check for auth token in environment
	token := os.Getenv("WRALE_AUTH_TOKEN")
	if token == "" {
		// Load from config if not in environment
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		if cfg.Token == "" {
			return nil, fmt.Errorf("no auth token configured - set WRALE_AUTH_TOKEN or authenticate using 'wsignctl login'")
		}
		token = cfg.Token
	}

	// Create client with loaded configuration
	c, err := client.NewClient(
		apiURL,
		client.WithToken(token),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return c, nil
}
