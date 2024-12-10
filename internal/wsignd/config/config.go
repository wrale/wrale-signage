// Package config provides configuration management for the Wrale Signage server
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the server
type Config struct {
	// Server contains HTTP server configuration
	Server ServerConfig
	// Database contains database connection configuration
	Database DatabaseConfig
	// Auth contains authentication configuration
	Auth AuthConfig
	// Content contains content delivery configuration
	Content ContentConfig
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	// Host is the server bind address
	Host string
	// Port is the server listening port
	Port int
	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration before timing out writes of the response
	WriteTimeout time.Duration
	// IdleTimeout is the maximum duration to wait for the next request
	IdleTimeout time.Duration
	// TLSCert is the path to the TLS certificate file
	TLSCert string
	// TLSKey is the path to the TLS private key file
	TLSKey string
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	// Host is the database server hostname
	Host string
	// Port is the database server port
	Port int
	// Name is the database name
	Name string
	// User is the database user
	User string
	// Password is the database password
	Password string
	// SSLMode is the PostgreSQL SSL mode
	SSLMode string
	// MaxOpenConns is the maximum number of open connections
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections
	MaxIdleConns int
	// ConnMaxLifetime is the maximum amount of time a connection may be reused
	ConnMaxLifetime time.Duration
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	// TokenSigningKey is the key used to sign JWT tokens
	TokenSigningKey string
	// TokenExpiry is how long tokens are valid for
	TokenExpiry time.Duration
	// DeviceCodeExpiry is how long device activation codes are valid
	DeviceCodeExpiry time.Duration
}

// ContentConfig holds content delivery settings
type ContentConfig struct {
	// StoragePath is where content files are stored
	StoragePath string
	// MaxCacheSize is the maximum size of the content cache in bytes
	MaxCacheSize int64
	// DefaultTTL is the default time-to-live for cached content
	DefaultTTL time.Duration
}

// Load creates a new Config from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Load server config
	cfg.Server = ServerConfig{
		Host:         getEnv("WSIGN_SERVER_HOST", "0.0.0.0"),
		Port:         getEnvAsInt("WSIGN_SERVER_PORT", 8080),
		ReadTimeout:  getEnvAsDuration("WSIGN_SERVER_READ_TIMEOUT", 5*time.Second),
		WriteTimeout: getEnvAsDuration("WSIGN_SERVER_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:  getEnvAsDuration("WSIGN_SERVER_IDLE_TIMEOUT", 120*time.Second),
		TLSCert:      getEnv("WSIGN_TLS_CERT", ""),
		TLSKey:       getEnv("WSIGN_TLS_KEY", ""),
	}

	// Load database config
	cfg.Database = DatabaseConfig{
		Host:            getEnv("WSIGN_DB_HOST", "localhost"),
		Port:            getEnvAsInt("WSIGN_DB_PORT", 5432),
		Name:            getEnv("WSIGN_DB_NAME", "wrale_signage"),
		User:            getEnv("WSIGN_DB_USER", "postgres"),
		Password:        getEnv("WSIGN_DB_PASSWORD", ""),
		SSLMode:         getEnv("WSIGN_DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvAsInt("WSIGN_DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("WSIGN_DB_MAX_IDLE_CONNS", 25),
		ConnMaxLifetime: getEnvAsDuration("WSIGN_DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}

	// Load auth config
	cfg.Auth = AuthConfig{
		TokenSigningKey:  getEnvRequired("WSIGN_AUTH_TOKEN_KEY"),
		TokenExpiry:      getEnvAsDuration("WSIGN_AUTH_TOKEN_EXPIRY", 1*time.Hour),
		DeviceCodeExpiry: getEnvAsDuration("WSIGN_AUTH_DEVICE_CODE_EXPIRY", 15*time.Minute),
	}

	// Load content config
	cfg.Content = ContentConfig{
		StoragePath:  getEnv("WSIGN_CONTENT_PATH", "/var/lib/wrale-signage/content"),
		MaxCacheSize: getEnvAsInt64("WSIGN_CONTENT_CACHE_SIZE", 1024*1024*1024), // 1GB
		DefaultTTL:   getEnvAsDuration("WSIGN_CONTENT_TTL", 1*time.Hour),
	}

	return cfg, cfg.validate()
}

// Validate checks that the configuration is valid
func (c *Config) validate() error {
	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// If TLS is configured, both cert and key must be provided
	if (c.Server.TLSCert != "") != (c.Server.TLSKey != "") {
		return fmt.Errorf("both TLS cert and key must be provided")
	}

	// Validate database config
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Database.MaxOpenConns < 1 {
		return fmt.Errorf("invalid max open connections: %d", c.Database.MaxOpenConns)
	}
	if c.Database.MaxIdleConns < 1 {
		return fmt.Errorf("invalid max idle connections: %d", c.Database.MaxIdleConns)
	}

	// Validate auth config
	if c.Auth.TokenSigningKey == "" {
		return fmt.Errorf("token signing key is required")
	}
	if c.Auth.TokenExpiry < 1*time.Minute {
		return fmt.Errorf("token expiry must be at least 1 minute")
	}

	// Validate content config
	if c.Content.MaxCacheSize < 1024*1024 { // 1MB minimum
		return fmt.Errorf("cache size must be at least 1MB")
	}

	return nil
}

// Helper functions for environment variable parsing

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvRequired(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	panic(fmt.Sprintf("required environment variable not set: %s", key))
}

func getEnvAsInt(key string, fallback int) int {
	if strValue, exists := os.LookupEnv(key); exists {
		if value, err := strconv.Atoi(strValue); err == nil {
			return value
		}
	}
	return fallback
}

func getEnvAsInt64(key string, fallback int64) int64 {
	if strValue, exists := os.LookupEnv(key); exists {
		if value, err := strconv.ParseInt(strValue, 10, 64); err == nil {
			return value
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if strValue, exists := os.LookupEnv(key); exists {
		if value, err := time.ParseDuration(strValue); err == nil {
			return value
		}
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if strValue, exists := os.LookupEnv(key); exists {
		if value, err := strconv.ParseBool(strValue); err == nil {
			return value
		}
	}
	return fallback
}

func getEnvAsSlice(key string, fallback []string, sep string) []string {
	if strValue, exists := os.LookupEnv(key); exists {
		return strings.Split(strValue, sep)
	}
	return fallback
}
