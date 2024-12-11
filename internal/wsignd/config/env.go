package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

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

// Basic environment variable helpers
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