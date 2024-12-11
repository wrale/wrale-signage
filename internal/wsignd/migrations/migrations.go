// Package migrations handles database schema management
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"time"
)

//go:embed *.sql
var migrationFiles embed.FS

var migrationFilePattern = regexp.MustCompile(`^(\d{3})_(.+)\.sql$`)

// Migration represents a single database migration
type Migration struct {
	// Version is a unique identifier for this migration
	Version int
	// Description provides context about what this migration does
	Description string
	// Up contains SQL statements for applying the migration
	Up string
	// Down contains SQL statements for reverting the migration
	Down string
}

// Manager handles executing database migrations
type Manager struct {
	db *sql.DB
}

// NewManager creates a new migration manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// LoadMigrations reads all SQL migration files
func (m *Manager) LoadMigrations() ([]Migration, error) {
	// Read embedded migration files
	entries, err := migrationFiles.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("error reading migrations: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		matches := migrationFilePattern.FindStringSubmatch(filename)
		if matches == nil {
			continue // Skip files that don't match pattern
		}

		// Parse version and description from filename
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid migration version in %s: %w", filename, err)
		}

		description := matches[2]

		// Read migration content
		content, err := migrationFiles.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("error reading migration %s: %w", filename, err)
		}

		migrations = append(migrations, Migration{
			Version:     version,
			Description: description,
			Up:         string(content),
			// Down migrations not currently supported
			Down: "",
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// ApplyMigrations runs any pending migrations
func (m *Manager) ApplyMigrations(ctx context.Context) error {
	// Ensure migration table exists
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("error creating migration table: %w", err)
	}

	// Load migrations
	migrations, err := m.LoadMigrations()
	if err != nil {
		return fmt.Errorf("error loading migrations: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("error getting applied migrations: %w", err)
	}

	// Apply pending migrations in order
	for _, migration := range migrations {
		if _, ok := applied[migration.Version]; !ok {
			if err := m.applyMigration(ctx, migration); err != nil {
				return fmt.Errorf("error applying migration %d: %w",
					migration.Version, err)
			}
		}
	}

	return nil
}

// ensureMigrationTable creates the migration tracking table if needed
func (m *Manager) ensureMigrationTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version        INTEGER PRIMARY KEY,
			applied_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			description   TEXT NOT NULL
		)
	`

	_, err := m.db.ExecContext(ctx, query)
	return err
}

// getAppliedMigrations returns a map of already applied migration versions
func (m *Manager) getAppliedMigrations(ctx context.Context) (map[int]time.Time, error) {
	query := `
		SELECT version, applied_at 
		FROM schema_migrations 
		ORDER BY version
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]time.Time)
	for rows.Next() {
		var version int
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		applied[version] = appliedAt
	}

	return applied, rows.Err()
}

// applyMigration executes a single migration within a transaction
func (m *Manager) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Error rolling back migration transaction: %v", err)
		}
	}()

	// Apply the migration
	if _, err := tx.ExecContext(ctx, migration.Up); err != nil {
		return err
	}

	// Record the migration
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (version, description)
		VALUES ($1, $2)
	`, migration.Version, migration.Description); err != nil {
		return err
	}

	return tx.Commit()
}