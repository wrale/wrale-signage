package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
)

// SessionParam represents a session parameter with its type for proper formatting
type SessionParam struct {
	Name  string
	Value string
	Type  ParamType
}

// ParamType indicates how a parameter should be formatted in SQL
type ParamType int

const (
	// ParamTypeEnum for parameters that need single quotes
	ParamTypeEnum ParamType = iota
	// ParamTypeDuration for parameters that need millisecond conversion
	ParamTypeDuration
)

// Session parameters for test database configuration
var testSessionParams = []SessionParam{
	{
		Name:  "default_transaction_isolation",
		Value: "serializable",
		Type:  ParamTypeEnum,
	},
	{
		Name:  "statement_timeout",
		Value: "5s",
		Type:  ParamTypeDuration,
	},
	{
		Name:  "lock_timeout",
		Value: "1s",
		Type:  ParamTypeDuration,
	},
	{
		Name:  "idle_in_transaction_session_timeout",
		Value: "1s",
		Type:  ParamTypeDuration,
	},
}

// SetupTestDB creates a test database connection and ensures it's ready
func SetupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		baseURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	// Connect to default postgres database first
	adminDB, err := tryConnect(t, baseURL)
	require.NoError(t, err, "Failed to connect to postgres database")
	defer adminDB.Close()

	// Create test database
	dbName := fmt.Sprintf("wrale_test_%d", time.Now().UnixNano())
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	require.NoError(t, err)

	// Connect to test database
	testURL := fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", dbName)
	db, err := tryConnect(t, testURL)
	require.NoError(t, err)

	// Configure session parameters for test environment
	err = configureTestSession(t, db)
	require.NoError(t, err)

	// Run migrations
	err = database.RunMigrations(db)
	require.NoError(t, err)

	cleanup := func() {
		if cerr := db.Close(); cerr != nil {
			t.Logf("Error closing test database connection: %v", cerr)
		}

		// Cleanup test database
		adminDB, err := sql.Open("postgres", baseURL)
		if err != nil {
			t.Logf("Error connecting to drop test database: %v", err)
			return
		}
		defer adminDB.Close()

		// Terminate existing connections
		_, err = adminDB.Exec(fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s'", dbName))
		if err != nil {
			t.Logf("Error terminating connections to test database: %v", err)
		}

		_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		if err != nil {
			t.Logf("Error dropping test database: %v", err)
		}
	}

	return db, cleanup
}

// formatParamValue formats a session parameter value according to its type
func formatParamValue(param SessionParam) (string, error) {
	switch param.Type {
	case ParamTypeEnum:
		// Enum values need single quotes
		return fmt.Sprintf("'%s'", param.Value), nil

	case ParamTypeDuration:
		// Convert duration string to milliseconds
		d, err := time.ParseDuration(param.Value)
		if err != nil {
			return "", fmt.Errorf("invalid duration %s: %w", param.Value, err)
		}
		return fmt.Sprintf("%d", d.Milliseconds()), nil

	default:
		return "", fmt.Errorf("unknown parameter type: %v", param.Type)
	}
}

// configureTestSession sets up optimal PostgreSQL session parameters for testing
func configureTestSession(t *testing.T, db *sql.DB) error {
	t.Helper()

	for _, param := range testSessionParams {
		value, err := formatParamValue(param)
		if err != nil {
			return fmt.Errorf("failed to format %s: %w", param.Name, err)
		}

		// Set session parameter
		_, err = db.Exec(fmt.Sprintf("SET SESSION %s = %s", param.Name, value))
		if err != nil {
			return fmt.Errorf("failed to set %s: %w", param.Name, err)
		}
	}

	// Verify configuration for debugging
	for _, param := range testSessionParams {
		var current string
		err := db.QueryRow(fmt.Sprintf("SHOW %s", param.Name)).Scan(&current)
		if err != nil {
			t.Logf("Warning: failed to verify %s setting: %v", param.Name, err)
			continue
		}
		t.Logf("Session parameter %s = %s", param.Name, current)
	}

	return nil
}

// tryConnect attempts to connect to database with retries
func tryConnect(t *testing.T, dbURL string) (*sql.DB, error) {
	t.Helper()

	var db *sql.DB
	var err error
	maxRetries := 5
	retryDelay := time.Second

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			t.Logf("Failed to open database connection (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}

		// Test connection
		err = db.Ping()
		if err == nil {
			break
		}
		t.Logf("Failed to ping database (attempt %d/%d): %v", i+1, maxRetries, err)
		if cerr := db.Close(); cerr != nil {
			t.Logf("Error closing failed connection: %v", cerr)
		}
		time.Sleep(retryDelay)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
	}

	return db, nil
}
