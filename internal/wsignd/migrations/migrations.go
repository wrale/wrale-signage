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
	"strings"
	"time"
)

//go:embed *.sql
var migrationFiles embed.FS

var (
	migrationFilePattern = regexp.MustCompile(`^(\d{3})_(.+)\.sql$`)
	functionPattern     = regexp.MustCompile(`(?si)CREATE(?:\s+OR\s+REPLACE)?\s+FUNCTION.*?LANGUAGE`)
)

// Migration represents a single database migration
type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
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
			continue
		}

		version, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid migration version in %s: %w", filename, err)
		}

		content, err := migrationFiles.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("error reading migration %s: %w", filename, err)
		}

		migrations = append(migrations, Migration{
			Version:     version,
			Description: matches[2],
			Up:         string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// ApplyMigrations runs any pending migrations
func (m *Manager) ApplyMigrations(ctx context.Context) error {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("error creating migration table: %w", err)
	}

	migrations, err := m.LoadMigrations()
	if err != nil {
		return fmt.Errorf("error loading migrations: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("error getting applied migrations: %w", err)
	}

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

// splitStatements splits SQL into individual statements while preserving functions
func (m *Manager) splitStatements(sql string) []string {
	// Extract function definitions first
	functions := functionPattern.FindAllString(sql, -1)
	for i, fn := range functions {
		sql = strings.Replace(sql, fn, fmt.Sprintf("--FUNCTION_%d--", i), 1)
	}

	// Split remaining SQL on semicolons
	statements := strings.Split(sql, ";")

	// Restore function definitions
	for i, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		for j, fn := range functions {
			statement = strings.Replace(statement, fmt.Sprintf("--FUNCTION_%d--", j), fn, 1)
		}
		statements[i] = statement
	}

	// Remove empty statements
	var result []string
	for _, stmt := range statements {
		if strings.TrimSpace(stmt) != "" {
			result = append(result, stmt)
		}
	}

	return result
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

	// Split and execute statements
	statements := m.splitStatements(migration.Up)
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("error executing statement: %w", err)
		}
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