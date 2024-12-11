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

var defaultConfig = Config{
	CurrentContext: "dev",
	Contexts: map[string]*Context{
		"dev": {
			Name:   "dev",
			Server: "http://localhost:8080",
			Token:  "dev-secret-key",
		},
	},
}

// defaultConfigPath returns the default config file path
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".wsignctl/config.yaml"
	}
	return filepath.Join(home, ".wsignctl/config.yaml")
}

// LoadConfig loads the configuration, using the provided path or falling back to defaults
func LoadConfig(configPath string) (*Config, error) {
	// Initialize viper
	v := viper.New()
	v.SetConfigType("yaml")

	// Try explicit config path first
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("error reading config from %s: %w", configPath, err)
		}
	} else {
		// Fall back to default paths
		defaultPath := defaultConfigPath()
		v.SetConfigFile(defaultPath)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// No config found anywhere, return defaults
				return &defaultConfig, nil
			}
			return nil, fmt.Errorf("error reading default config: %w", err)
		}
	}

	// Parse config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &config, nil
}

// SaveConfig writes the configuration to disk
func SaveConfig(config *Config, configPath string) error {
	v := viper.New()
	v.SetConfigType("yaml")

	// Determine config path
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	v.SetConfigFile(configPath)

	// Update config values
	v.Set("current-context", config.CurrentContext)
	v.Set("contexts", config.Contexts)

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Write config
	if err := v.WriteConfig(); err != nil {
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