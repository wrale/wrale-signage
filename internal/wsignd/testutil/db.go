package testutil

import (
	"database/sql"
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

	// Allow overriding test database URL through environment
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/wrale_test?sslmode=disable"
	}

	// Try connecting with retry logic
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
			t.Logf("Failed to close failed connection: %v", cerr)
		}
		time.Sleep(retryDelay)
	}

	require.NoError(t, err, "Failed to connect to test database after %d attempts", maxRetries)

	// Ensure we have our test database
	_, err = db.Exec("DROP DATABASE IF EXISTS wrale_test")
	require.NoError(t, err)
	_, err = db.Exec("CREATE DATABASE wrale_test")
	require.NoError(t, err)

	// Reconnect to test database
	db, err = sql.Open("postgres", dbURL)
	require.NoError(t, err)

	// Run migrations
	err = database.RunMigrations(db)
	require.NoError(t, err)

	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close test database: %v", err)
		}
	}

	return db, cleanup
}
