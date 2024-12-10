// Package config provides configuration management for the Wrale Signage CLI
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the CLI configuration
type Config struct {
	// CurrentContext is the name of the active context
	CurrentContext string `mapstructure:"current-context"`
	// Contexts holds the available server contexts
	Contexts map[string]*Context `mapstructure:"contexts"`
}

// Context represents a server configuration context
type Context struct {
	// Name is the context identifier
	Name string `mapstructure:"name"`
	// Server is the API server URL
	Server string `mapstructure:"server"`
	// Token is the authentication token
	Token string `mapstructure:"token"`
	// InsecureSkipVerify disables TLS verification
	InsecureSkipVerify bool `mapstructure:"insecure-skip-verify"`
}

// defaultConfigPath returns the default config file path
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".wsignctl/config.yaml"
	}
	return filepath.Join(home, ".wsignctl/config.yaml")
}

// LoadConfig loads the configuration from disk
func LoadConfig() (*Config, error) {
	configPath := os.Getenv("WSIGNCTL_CONFIG")
	if configPath == "" {
		configPath = defaultConfigPath()
	}

	// Initialize viper with default values
	viper.SetDefault("current-context", "")
	viper.SetDefault("contexts", map[string]*Context{})

	// Set up viper to read the config file
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Try to read the config file
	if err := viper.ReadInConfig(); err != nil {
		// If the config file doesn't exist, create a default one
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Ensure directory exists
			configDir := filepath.Dir(configPath)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return nil, fmt.Errorf("error creating config directory: %w", err)
			}

			// Write default config
			if err := viper.SafeWriteConfig(); err != nil {
				return nil, fmt.Errorf("error writing default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	// Parse the config into our struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &config, nil
}

// SaveConfig writes the configuration to disk
func SaveConfig(config *Config) error {
	// Update viper with new config
	if err := viper.Set("current-context", config.CurrentContext); err != nil {
		return fmt.Errorf("error setting current-context: %w", err)
	}
	if err := viper.Set("contexts", config.Contexts); err != nil {
		return fmt.Errorf("error setting contexts: %w", err)
	}

	// Write to disk
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("error writing config: %w", err)
	}

	return nil
}

// GetCurrentContext returns the active context configuration
func (c *Config) GetCurrentContext() (*Context, error) {
	if c.CurrentContext == "" {
		return nil, fmt.Errorf("no current context set")
	}

	ctx, ok := c.Contexts[c.CurrentContext]
	if !ok {
		return nil, fmt.Errorf("current context %q not found", c.CurrentContext)
	}

	return ctx, nil
}

// AddContext adds or updates a context in the configuration
func (c *Config) AddContext(name string, context *Context) {
	if c.Contexts == nil {
		c.Contexts = make(map[string]*Context)
	}
	context.Name = name
	c.Contexts[name] = context
}

// SetCurrentContext sets the active context
func (c *Config) SetCurrentContext(name string) error {
	if _, ok := c.Contexts[name]; !ok {
		return fmt.Errorf("context %q not found", name)
	}
	c.CurrentContext = name
	return nil
}

// RemoveContext removes a context from the configuration
func (c *Config) RemoveContext(name string) error {
	if _, ok := c.Contexts[name]; !ok {
		return fmt.Errorf("context %q not found", name)
	}
	delete(c.Contexts, name)

	// If we removed the current context, clear it
	if c.CurrentContext == name {
		c.CurrentContext = ""
	}

	return nil
}
