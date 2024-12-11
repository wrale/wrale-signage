// Package config provides configuration management for the Wrale Signage server
package config

import (
	"time"
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
	Port         int          `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	IdleTimeout  time.Duration `yaml:"idleTimeout"`
	TLSCert      string       `yaml:"tlsCert"`
	TLSKey       string       `yaml:"tlsKey"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int          `yaml:"port"`
	Name            string       `yaml:"name"`
	User            string       `yaml:"user"`
	Password        string       `yaml:"password"`
	SSLMode         string       `yaml:"sslmode"`
	MaxOpenConns    int          `yaml:"maxOpenConns"`
	MaxIdleConns    int          `yaml:"maxIdleConns"`
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

// overlayEnv overlays environment variables on top of file-based config
func (c *Config) overlayEnv() {
	// Server config
	if host := getEnv("WSIGN_SERVER_HOST", ""); host != "" {
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
	if tlsCert := getEnv("WSIGN_TLS_CERT", ""); tlsCert != "" {
		c.Server.TLSCert = tlsCert
	}
	if tlsKey := getEnv("WSIGN_TLS_KEY", ""); tlsKey != "" {
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
	if sslmode := getEnv("WSIGN_DB_SSLMODE", ""); sslmode != "" {
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
	if key := getEnv("WSIGN_AUTH_TOKEN_KEY", ""); key != "" {
		c.Auth.TokenSigningKey = key
	}
	if expiry := getEnvAsDuration("WSIGN_AUTH_TOKEN_EXPIRY", 0); expiry != 0 {
		c.Auth.TokenExpiry = expiry
	}
	if deviceExpiry := getEnvAsDuration("WSIGN_AUTH_DEVICE_CODE_EXPIRY", 0); deviceExpiry != 0 {
		c.Auth.DeviceCodeExpiry = deviceExpiry
	}

	// Content config
	if path := getEnv("WSIGN_CONTENT_PATH", ""); path != "" {
		c.Content.StoragePath = path
	}
	if size := getEnvAsInt64("WSIGN_CONTENT_CACHE_SIZE", 0); size != 0 {
		c.Content.MaxCacheSize = size
	}
	if ttl := getEnvAsDuration("WSIGN_CONTENT_TTL", 0); ttl != 0 {
		c.Content.DefaultTTL = ttl
	}
}