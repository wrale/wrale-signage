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
