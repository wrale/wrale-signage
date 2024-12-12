// Package postgres provides PostgreSQL implementations of the content domain repositories.
package postgres

import (
	"database/sql"

	"github.com/wrale/wrale-signage/internal/wsignd/content"
)

// repository implements the content.Repository interface using PostgreSQL.
type repository struct {
	db *sql.DB
}

// NewRepository creates a new PostgreSQL-backed content repository.
func NewRepository(db *sql.DB) content.Repository {
	return &repository{db: db}
}
