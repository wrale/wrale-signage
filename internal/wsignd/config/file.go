package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	// DefaultConfigDirs lists the allowed configuration directories in order of preference
	DefaultConfigDirs = []string{
		"/etc/wrale-signage",
		"/usr/local/etc/wrale-signage",
		"/users/josh/src/wrale-signage", // Development path
	}

	// allowedExtensions lists the allowed config file extensions
	allowedExtensions = []string{".yaml", ".yml"}
)

// validateConfigPath ensures the config file path is secure
func validateConfigPath(path string) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid config path: %w", err)
	}

	// Clean the path
	cleanPath := filepath.Clean(absPath)

	// Resolve any symlinks
	realPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("error resolving config path: %w", err)
		}
		// If file doesn't exist yet, use the cleaned path
		realPath = cleanPath
	}

	// Verify extension
	validExt := false
	for _, ext := range allowedExtensions {
		if strings.HasSuffix(strings.ToLower(realPath), ext) {
			validExt = true
			break
		}
	}
	if !validExt {
		return "", fmt.Errorf("config file must have .yaml or .yml extension")
	}

	// Check if path is within allowed directories
	validPath := false
	configRoot := filepath.Dir(realPath)
	for _, dir := range DefaultConfigDirs {
		if strings.HasPrefix(strings.ToLower(configRoot), strings.ToLower(dir)) {
			validPath = true
			break
		}
	}

	// Also allow paths relative to current directory in development
	if !validPath && os.Getenv("WSIGN_DEV_MODE") == "1" {
		pwd, err := os.Getwd()
		if err == nil {
			validPath = strings.HasPrefix(configRoot, pwd)
		}
	}

	if !validPath {
		return "", fmt.Errorf("config file must be in an allowed directory (tried: %s)", configRoot)
	}

	return realPath, nil
}

// safeReadFile reads a file that has been validated as safe
// This function should only be used with paths that have passed through validateConfigPath
func safeReadFile(path string) ([]byte, error) {
	// Verify the file exists and is a regular file
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("error accessing config file: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("config path must be a regular file")
	}

	// #nosec G304 -- path has been validated by validateConfigPath
	return os.ReadFile(path)
}

// LoadFile loads configuration from a YAML file
func LoadFile(path string) (*Config, error) {
	// Validate and resolve the config path
	validPath, err := validateConfigPath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid config path: %w", err)
	}

	// Read the validated config file
	data, err := safeReadFile(validPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Overlay environment variables
	cfg.overlayEnv()

	return &cfg, cfg.validate()
}
