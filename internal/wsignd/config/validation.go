package config

import (
	"fmt"
	"time"
)

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
