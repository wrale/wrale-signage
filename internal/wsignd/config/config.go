// Package config provides configuration management for the Wrale Signage server
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the server
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Content  ContentConfig  `yaml:"content"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	IdleTimeout  time.Duration `yaml:"idleTimeout"`
	TLSCert      string        `yaml:"tlsCert"`
	TLSKey       string        `yaml:"tlsKey"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Name            string        `yaml:"name"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"maxOpenConns"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"`
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	TokenSigningKey  string        `yaml:"tokenSigningKey"`
	TokenExpiry      time.Duration `yaml:"tokenExpiry"`
	DeviceCodeExpiry time.Duration `yaml:"deviceCodeExpiry"`
}

// ContentConfig holds content delivery settings
type ContentConfig struct {
	StoragePath  string        `yaml:"storagePath"`
	MaxCacheSize int64         `yaml:"maxCacheSize"`
	DefaultTTL   time.Duration `yaml:"defaultTTL"`
}

// LoadFile loads configuration from a YAML file
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Overlay environment variables
	cfg.overlayEnv()

	return &cfg, cfg.validate()
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

	// Load database config with both WSIGN_ and direct env vars
	cfg.Database = DatabaseConfig{
		Host:            getEnvMulti([]string{"WSIGN_DB_HOST", "DB_HOST", "POSTGRES_HOST"}, "localhost"),
		Port:            getEnvAsIntMulti([]string{"WSIGN_DB_PORT", "DB_PORT", "POSTGRES_PORT"}, 5432),
		Name:            getEnvMulti([]string{"WSIGN_DB_NAME", "DB_NAME", "POSTGRES_DB"}, "wrale_signage"),
		User:            getEnvMulti([]string{"WSIGN_DB_USER", "DB_USER", "POSTGRES_USER"}, "postgres"),
		Password:        getEnvMulti([]string{"WSIGN_DB_PASSWORD", "DB_PASSWORD", "POSTGRES_PASSWORD"}, ""),
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

// overlayEnv overlays environment variables on top of file-based config
func (c *Config) overlayEnv() {
	// Server config
	if host := os.Getenv("WSIGN_SERVER_HOST"); host != "" {
		c.Server.Host = host
	}
	if port := getEnvAsInt("WSIGN_SERVER_PORT", 0); port != 0 {
		c.Server.Port = port
	}
	if readTimeout := getEnvAsDuration("WSIGN_SERVER_READ_TIMEOUT", 0); readTimeout != 0 {
		c.Server.ReadTimeout = readTimeout
	}
	if writeTimeout := getEnvAsDuration("WSIGN_SERVER_WRITE_TIMEOUT", 0); writeTimeout != 0 {
		c.Server.WriteTimeout = writeTimeout
	}
	if idleTimeout := getEnvAsDuration("WSIGN_SERVER_IDLE_TIMEOUT", 0); idleTimeout != 0 {
		c.Server.IdleTimeout = idleTimeout
	}
	if tlsCert := os.Getenv("WSIGN_TLS_CERT"); tlsCert != "" {
		c.Server.TLSCert = tlsCert
	}
	if tlsKey := os.Getenv("WSIGN_TLS_KEY"); tlsKey != "" {
		c.Server.TLSKey = tlsKey
	}

	// Database config - check multiple env var names
	if host := getEnvMulti([]string{"WSIGN_DB_HOST", "DB_HOST", "POSTGRES_HOST"}, ""); host != "" {
		c.Database.Host = host
	}
	if port := getEnvAsIntMulti([]string{"WSIGN_DB_PORT", "DB_PORT", "POSTGRES_PORT"}, 0); port != 0 {
		c.Database.Port = port
	}
	if name := getEnvMulti([]string{"WSIGN_DB_NAME", "DB_NAME", "POSTGRES_DB"}, ""); name != "" {
		c.Database.Name = name
	}
	if user := getEnvMulti([]string{"WSIGN_DB_USER", "DB_USER", "POSTGRES_USER"}, ""); user != "" {
		c.Database.User = user
	}
	if password := getEnvMulti([]string{"WSIGN_DB_PASSWORD", "DB_PASSWORD", "POSTGRES_PASSWORD"}, ""); password != "" {
		c.Database.Password = password
	}
	if sslmode := os.Getenv("WSIGN_DB_SSLMODE"); sslmode != "" {
		c.Database.SSLMode = sslmode
	}
	if maxOpenConns := getEnvAsInt("WSIGN_DB_MAX_OPEN_CONNS", 0); maxOpenConns != 0 {
		c.Database.MaxOpenConns = maxOpenConns
	}
	if maxIdleConns := getEnvAsInt("WSIGN_DB_MAX_IDLE_CONNS", 0); maxIdleConns != 0 {
		c.Database.MaxIdleConns = maxIdleConns
	}
	if connMaxLifetime := getEnvAsDuration("WSIGN_DB_CONN_MAX_LIFETIME", 0); connMaxLifetime != 0 {
		c.Database.ConnMaxLifetime = connMaxLifetime
	}

	// Auth config
	if key := os.Getenv("WSIGN_AUTH_TOKEN_KEY"); key != "" {
		c.Auth.TokenSigningKey = key
	}
	if expiry := getEnvAsDuration("WSIGN_AUTH_TOKEN_EXPIRY", 0); expiry != 0 {
		c.Auth.TokenExpiry = expiry
	}
	if deviceExpiry := getEnvAsDuration("WSIGN_AUTH_DEVICE_CODE_EXPIRY", 0); deviceExpiry != 0 {
		c.Auth.DeviceCodeExpiry = deviceExpiry
	}

	// Content config
	if path := os.Getenv("WSIGN_CONTENT_PATH"); path != "" {
		c.Content.StoragePath = path
	}
	if size := getEnvAsInt64("WSIGN_CONTENT_CACHE_SIZE", 0); size != 0 {
		c.Content.MaxCacheSize = size
	}
	if ttl := getEnvAsDuration("WSIGN_CONTENT_TTL", 0); ttl != 0 {
		c.Content.DefaultTTL = ttl
	}
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

// getEnvMulti tries multiple environment variables in order
func getEnvMulti(keys []string, fallback string) string {
	for _, key := range keys {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
	}
	return fallback
}

// getEnvAsIntMulti tries multiple environment variables in order
func getEnvAsIntMulti(keys []string, fallback int) int {
	for _, key := range keys {
		if strValue, exists := os.LookupEnv(key); exists {
			if value, err := strconv.Atoi(strValue); err == nil {
				return value
			}
		}
	}
	return fallback
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
