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
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Content  ContentConfig
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	TLSCert      string
	TLSKey       string
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	TokenSigningKey  string
	TokenExpiry      time.Duration
	DeviceCodeExpiry time.Duration
}

// ContentConfig holds content delivery settings
type ContentConfig struct {
	StoragePath  string
	MaxCacheSize int64
	DefaultTTL   time.Duration
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

func (c *Config) validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if (c.Server.TLSCert != "") != (c.Server.TLSKey != "") {
		return fmt.Errorf("both TLS cert and key must be provided")
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Database.MaxOpenConns < 1 {
		return fmt.Errorf("invalid max open connections: %d", c.Database.MaxOpenConns)
	}
	if c.Database.MaxIdleConns < 1 {
		return fmt.Errorf("invalid max idle connections: %d", c.Database.MaxIdleConns)
	}
	if c.Auth.TokenSigningKey == "" {
		return fmt.Errorf("token signing key is required")
	}
	if c.Auth.TokenExpiry < 1*time.Minute {
		return fmt.Errorf("token expiry must be at least 1 minute")
	}
	if c.Content.MaxCacheSize < 1024*1024 { // 1MB minimum
		return fmt.Errorf("cache size must be at least 1MB")
	}
	return nil
}

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

// getEnvAsBool parses a boolean environment variable with fallback
// nolint:unused // Reserved for future use
func getEnvAsBool(key string, fallback bool) bool {
	if strValue, exists := os.LookupEnv(key); exists {
		if value, err := strconv.ParseBool(strValue); err == nil {
			return value
		}
	}
	return fallback
}

// getEnvAsSlice splits an environment variable into a slice with fallback
// nolint:unused // Reserved for future use
func getEnvAsSlice(key string, fallback []string, sep string) []string {
	if strValue, exists := os.LookupEnv(key); exists {
		return strings.Split(strValue, sep)
	}
	return fallback
}
