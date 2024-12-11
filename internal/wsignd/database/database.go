// Package database provides utilities for database operations
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
	"github.com/wrale/wrale-signage/internal/wsignd/migrations"
)

// Tx wraps a database transaction with additional functionality
type Tx struct {
	*sql.Tx
}

// TxOptions defines options for transaction execution
type TxOptions struct {
	// Isolation sets the transaction isolation level
	Isolation sql.IsolationLevel
	// ReadOnly indicates if the transaction is read-only
	ReadOnly bool
}

// SetupDatabase creates a connection pool and ensures migrations are applied
func SetupDatabase(connStr string, maxRetries int, backoff time.Duration) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			// Configure connection pool
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(25)
			db.SetConnMaxLifetime(5 * time.Minute)

			// Verify connection and run migrations
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := db.PingContext(ctx); err == nil {
				cancel()
				// Run migrations
				if err := RunMigrations(db); err != nil {
					db.Close()
					return nil, fmt.Errorf("migration failed: %w", err)
				}
				return db, nil
			}
			cancel()
			db.Close()
		}

		// Wait before retry
		time.Sleep(backoff)
		backoff *= 2 // Exponential backoff
	}

	return nil, fmt.Errorf("failed to setup database after %d attempts: %w", maxRetries, err)
}

// RunMigrations executes all SQL migrations using the migration manager
func RunMigrations(db *sql.DB) error {
	manager := migrations.NewManager(db)
	if err := manager.ApplyMigrations(context.Background()); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

// RunInTx executes a function within a transaction
func RunInTx(ctx context.Context, db *sql.DB, opts *TxOptions, fn func(*Tx) error) error {
	// Start transaction with proper options
	var txOpts *sql.TxOptions
	if opts != nil {
		txOpts = &sql.TxOptions{
			Isolation: opts.Isolation,
			ReadOnly:  opts.ReadOnly,
		}
	}

	tx, err := db.BeginTx(ctx, txOpts)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}

	// Wrap the sql.Tx with our custom Tx
	wtx := &Tx{Tx: tx}

	// Execute the provided function
	if err := fn(wtx); err != nil {
		// Attempt rollback on error
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("error rolling back transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// MapError converts database-specific errors to domain errors
func MapError(err error, op string) error {
	if err == nil {
		return nil
	}

	// Handle specific PostgreSQL errors
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505": // unique_violation
			if strings.Contains(pqErr.Error(), "displays_pkey") {
				return werrors.NewError(
					"CONFLICT",
					"display ID already exists",
					op,
					werrors.ErrConflict,
				)
			}
			if strings.Contains(pqErr.Error(), "displays_name_key") {
				return werrors.NewError(
					"DISPLAY_EXISTS",
					fmt.Sprintf("display already exists: %s", strings.TrimPrefix(pqErr.Message, "Key (name)=(")),
					op,
					werrors.ErrConflict,
				)
			}
			return werrors.NewError(
				"CONFLICT",
				"resource already exists",
				op,
				werrors.ErrConflict,
			)
		case "23503": // foreign_key_violation
			return werrors.NewError(
				"NOT_FOUND",
				"referenced resource not found",
				op,
				werrors.ErrNotFound,
			)
		case "23514": // check_violation
			return werrors.NewError(
				"INVALID_INPUT",
				pqErr.Message,
				op,
				werrors.ErrInvalidInput,
			)
		}
	}

	// Handle sql.ErrNoRows
	if errors.Is(err, sql.ErrNoRows) {
		return werrors.NewError(
			"NOT_FOUND",
			"resource not found",
			op,
			werrors.ErrNotFound,
		)
	}

	// Map other errors as internal errors
	return werrors.NewError(
		"INTERNAL",
		"internal database error",
		op,
		err,
	)
}

// GenerateInsertQuery creates an INSERT query with properly numbered placeholders
func GenerateInsertQuery(table string, columns []string) string {
	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
}

// GenerateUpdateQuery creates an UPDATE query with properly numbered placeholders
func GenerateUpdateQuery(table string, columns []string, whereColumns []string) string {
	// Create SET clause
	setItems := make([]string, len(columns))
	for i, col := range columns {
		setItems[i] = fmt.Sprintf("%s = $%d", col, i+1)
	}

	// Create WHERE clause
	whereItems := make([]string, len(whereColumns))
	for i, col := range whereColumns {
		whereItems[i] = fmt.Sprintf("%s = $%d", col, len(columns)+i+1)
	}

	return fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setItems, ", "),
		strings.Join(whereItems, " AND "),
	)
}

// ExecuteNamedQuery executes a query with named parameters
func ExecuteNamedQuery(ctx context.Context, tx *Tx, query string, params map[string]interface{}) (sql.Result, error) {
	// Replace named parameters with positional ones
	query, args := convertNamedParams(query, params)

	return tx.ExecContext(ctx, query, args...)
}

// convertNamedParams converts a query with named parameters to positional parameters
func convertNamedParams(query string, params map[string]interface{}) (string, []interface{}) {
	args := make([]interface{}, 0, len(params))
	paramNum := 1

	// Replace each :param with $N
	for name, value := range params {
		placeholder := ":" + name
		query = strings.Replace(query, placeholder, fmt.Sprintf("$%d", paramNum), -1)
		args = append(args, value)
		paramNum++
	}

	return query, args
}
