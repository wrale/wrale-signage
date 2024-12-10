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
	var token string

	if apiURL == "" || os.Getenv("WRALE_AUTH_TOKEN") == "" {
		// Load configuration if either URL or token not in environment
		cfg, err := config.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}

		// Get current context
		ctx, err := cfg.GetCurrentContext()
		if err != nil {
			return nil, fmt.Errorf("failed to get current context: %w", err)
		}

		// Use context values if environment variables not set
		if apiURL == "" {
			if ctx.Server == "" {
				return nil, fmt.Errorf("no API server configured - set WRALE_API_URL or configure server in wsignctl config")
			}
			apiURL = ctx.Server
		}

		if os.Getenv("WRALE_AUTH_TOKEN") == "" {
			if ctx.Token == "" {
				return nil, fmt.Errorf("no auth token configured - set WRALE_AUTH_TOKEN or authenticate using 'wsignctl login'")
			}
			token = ctx.Token
		}
	} else {
		// Use environment token if set
		token = os.Getenv("WRALE_AUTH_TOKEN")
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
