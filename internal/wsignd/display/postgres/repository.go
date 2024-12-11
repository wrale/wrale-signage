// Package postgres implements the display repository using PostgreSQL
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/wrale/wrale-signage/internal/wsignd/database"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

// Repository implements the display.Repository interface using PostgreSQL. It provides
// persistent storage for display entities while maintaining consistency through
// optimistic locking and proper transaction management.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new PostgreSQL display repository that fulfills the
// display.Repository interface contract.
func NewRepository(db *sql.DB) display.Repository {
	return &Repository{db: db}
}

// Save persists a display to the database, handling both creation and updates.
// It uses optimistic locking to prevent concurrent modifications and maintains
// data consistency through transactions.
func (r *Repository) Save(ctx context.Context, d *display.Display) error {
	const op = "DisplayRepository.Save"

	// Convert properties map to JSON for storage
	properties, err := json.Marshal(d.Properties)
	if err != nil {
		return fmt.Errorf("error marshaling properties: %w", err)
	}

	// Handle upsert with optimistic locking within a transaction
	err = database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		// Check if display exists
		var exists bool
		err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM displays WHERE id = $1
			)
		`, d.ID).Scan(&exists)
		if err != nil {
			return err
		}

		if exists {
			// Update existing display with version check for optimistic locking
			result, err := tx.ExecContext(ctx, `
				UPDATE displays 
				SET name = $1,
					site_id = $2,
					zone = $3,
					position = $4,
					state = $5,
					last_seen = $6,
					version = $7,
					properties = $8
				WHERE id = $9
				  AND version = $10
			`,
				d.Name,
				d.Location.SiteID,
				d.Location.Zone,
				d.Location.Position,
				d.State,
				d.LastSeen,
				d.Version+1,
				properties,
				d.ID,
				d.Version,
			)
			if err != nil {
				return err
			}

			rows, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if rows == 0 {
				return display.ErrVersionMismatch{ID: d.ID.String()}
			}

			// Update the version number on successful update
			d.Version++
		} else {
			// Insert new display record
			_, err = tx.ExecContext(ctx, `
				INSERT INTO displays (
					id, name, site_id, zone, position,
					state, last_seen, version, properties
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`,
				d.ID,
				d.Name,
				d.Location.SiteID,
				d.Location.Zone,
				d.Location.Position,
				d.State,
				d.LastSeen,
				d.Version,
				properties,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return database.MapError(err, op)
	}

	return nil
}

// FindByID retrieves a display by its unique identifier. It returns ErrNotFound
// if no display exists with the given ID.
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*display.Display, error) {
	const op = "DisplayRepository.FindByID"

	var d display.Display
	var propertiesJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, name, site_id, zone, position,
			state, last_seen, version, properties
		FROM displays
		WHERE id = $1
	`, id).Scan(
		&d.ID,
		&d.Name,
		&d.Location.SiteID,
		&d.Location.Zone,
		&d.Location.Position,
		&d.State,
		&d.LastSeen,
		&d.Version,
		&propertiesJSON,
	)
	if err != nil {
		return nil, database.MapError(err, op)
	}

	// Parse the JSON properties into the map
	if err := json.Unmarshal(propertiesJSON, &d.Properties); err != nil {
		return nil, fmt.Errorf("error unmarshaling properties: %w", err)
	}

	return &d, nil
}

// FindByName retrieves a display by its name, which must be unique across the system.
// It returns ErrNotFound if no display exists with the given name.
func (r *Repository) FindByName(ctx context.Context, name string) (*display.Display, error) {
	const op = "DisplayRepository.FindByName"

	var d display.Display
	var propertiesJSON []byte

	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, name, site_id, zone, position,
			state, last_seen, version, properties
		FROM displays
		WHERE name = $1
	`, name).Scan(
		&d.ID,
		&d.Name,
		&d.Location.SiteID,
		&d.Location.Zone,
		&d.Location.Position,
		&d.State,
		&d.LastSeen,
		&d.Version,
		&propertiesJSON,
	)
	if err != nil {
		return nil, database.MapError(err, op)
	}

	if err := json.Unmarshal(propertiesJSON, &d.Properties); err != nil {
		return nil, fmt.Errorf("error unmarshaling properties: %w", err)
	}

	return &d, nil
}

// List retrieves displays matching the provided filter criteria. It returns
// an empty slice if no matching displays are found.
func (r *Repository) List(ctx context.Context, filter display.DisplayFilter) ([]*display.Display, error) {
	const op = "DisplayRepository.List"

	// Build query with dynamic WHERE clause based on filter
	query := `
		SELECT 
			id, name, site_id, zone, position,
			state, last_seen, version, properties
		FROM displays
		WHERE 1=1
	`
	var args []interface{}
	var conditions []string

	// Add filter conditions
	if filter.SiteID != "" {
		args = append(args, filter.SiteID)
		conditions = append(conditions, fmt.Sprintf("site_id = $%d", len(args)))
	}
	if filter.Zone != "" {
		args = append(args, filter.Zone)
		conditions = append(conditions, fmt.Sprintf("zone = $%d", len(args)))
	}
	if len(filter.States) > 0 {
		args = append(args, filter.States)
		conditions = append(conditions, fmt.Sprintf("state = ANY($%d)", len(args)))
	}

	// Apply conditions to query
	for _, cond := range conditions {
		query += " AND " + cond
	}

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, database.MapError(err, op)
	}
	defer rows.Close()

	// Collect results
	var displays []*display.Display
	for rows.Next() {
		var d display.Display
		var propertiesJSON []byte

		err := rows.Scan(
			&d.ID,
			&d.Name,
			&d.Location.SiteID,
			&d.Location.Zone,
			&d.Location.Position,
			&d.State,
			&d.LastSeen,
			&d.Version,
			&propertiesJSON,
		)
		if err != nil {
			return nil, database.MapError(err, op)
		}

		if err := json.Unmarshal(propertiesJSON, &d.Properties); err != nil {
			return nil, fmt.Errorf("error unmarshaling properties: %w", err)
		}

		displays = append(displays, &d)
	}

	if err := rows.Err(); err != nil {
		return nil, database.MapError(err, op)
	}

	return displays, nil
}

// Delete removes a display from storage by its ID. It returns ErrNotFound
// if no display exists with the given ID.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayRepository.Delete"

	result, err := r.db.ExecContext(ctx, `
		DELETE FROM displays
		WHERE id = $1
	`, id)
	if err != nil {
		return database.MapError(err, op)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return database.MapError(err, op)
	}

	if rows == 0 {
		return database.MapError(sql.ErrNoRows, op)
	}

	return nil
}
