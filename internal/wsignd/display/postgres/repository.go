// Package postgres implements the display repository using PostgreSQL
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/wrale/wrale-signage/internal/wsignd/database"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

// Repository implements the display.Repository interface using PostgreSQL
type Repository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewRepository creates a new PostgreSQL display repository
func NewRepository(db *sql.DB, logger *slog.Logger) display.Repository {
	return &Repository{db: db, logger: logger}
}

// Save persists a display to the database, handling both creation and updates
func (r *Repository) Save(ctx context.Context, d *display.Display) error {
	const op = "DisplayRepository.Save"

	// Convert properties to JSON
	properties, err := json.Marshal(d.Properties)
	if err != nil {
		r.logger.Error("failed to marshal properties",
			"error", err,
			"displayID", d.ID,
			"operation", op,
		)
		return fmt.Errorf("error marshaling properties: %w", err)
	}

	err = database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		// First verify display exists for updates
		var currentVersion int
		err := tx.QueryRowContext(ctx, `
			SELECT version FROM displays WHERE id = $1
		`, d.ID).Scan(&currentVersion)

		if err != nil {
			if err == sql.ErrNoRows {
				r.logger.Info("creating new display",
					"displayID", d.ID,
					"name", d.Name,
					"state", d.State,
					"operation", op,
				)
				// Insert new display
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
					r.logger.Error("failed to insert display",
						"error", err,
						"displayID", d.ID,
						"operation", op,
					)
					return err
				}
				return nil
			}
			r.logger.Error("failed to check display version",
				"error", err,
				"displayID", d.ID,
				"operation", op,
			)
			return err
		}

		r.logger.Info("updating display",
			"displayID", d.ID,
			"name", d.Name,
			"currentVersion", currentVersion,
			"newVersion", d.Version,
			"newState", d.State,
			"operation", op,
		)

		// Verify version for optimistic locking
		if currentVersion != d.Version {
			r.logger.Warn("version mismatch",
				"displayID", d.ID,
				"currentVersion", currentVersion,
				"expectedVersion", d.Version,
				"operation", op,
			)
			return display.ErrVersionMismatch{ID: d.ID.String()}
		}

		// Update existing display
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
			r.logger.Error("failed to update display",
				"error", err,
				"displayID", d.ID,
				"operation", op,
			)
			return err
		}

		rows, err := result.RowsAffected()
		if err != nil {
			r.logger.Error("failed to get rows affected",
				"error", err,
				"displayID", d.ID,
				"operation", op,
			)
			return err
		}
		if rows == 0 {
			r.logger.Error("display not found during update",
				"displayID", d.ID,
				"operation", op,
			)
			return display.ErrNotFound{ID: d.ID.String()}
		}

		// Update version on success
		d.Version++

		r.logger.Info("successfully updated display",
			"displayID", d.ID,
			"name", d.Name,
			"newVersion", d.Version,
			"newState", d.State,
			"operation", op,
		)
		return nil
	})

	if err != nil {
		r.logger.Error("failed to save display",
			"error", err,
			"displayID", d.ID,
			"operation", op,
		)
		return database.MapError(err, op)
	}

	return nil
}

// FindByID retrieves a display by its unique identifier
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
		if err == sql.ErrNoRows {
			r.logger.Warn("display not found",
				"displayID", id,
				"operation", op,
			)
		} else {
			r.logger.Error("failed to find display",
				"error", err,
				"displayID", id,
				"operation", op,
			)
		}
		return nil, database.MapError(err, op)
	}

	if err := json.Unmarshal(propertiesJSON, &d.Properties); err != nil {
		r.logger.Error("failed to unmarshal properties",
			"error", err,
			"displayID", id,
			"operation", op,
		)
		return nil, fmt.Errorf("error unmarshaling properties: %w", err)
	}

	r.logger.Info("found display",
		"displayID", id,
		"name", d.Name,
		"state", d.State,
		"version", d.Version,
		"operation", op,
	)
	return &d, nil
}

// FindByName retrieves a display by its name
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
		if err == sql.ErrNoRows {
			r.logger.Warn("display not found by name",
				"name", name,
				"operation", op,
			)
		} else {
			r.logger.Error("failed to find display by name",
				"error", err,
				"name", name,
				"operation", op,
			)
		}
		return nil, database.MapError(err, op)
	}

	if err := json.Unmarshal(propertiesJSON, &d.Properties); err != nil {
		r.logger.Error("failed to unmarshal properties",
			"error", err,
			"name", name,
			"operation", op,
		)
		return nil, fmt.Errorf("error unmarshaling properties: %w", err)
	}

	r.logger.Info("found display by name",
		"name", name,
		"displayID", d.ID,
		"state", d.State,
		"version", d.Version,
		"operation", op,
	)
	return &d, nil
}

// List retrieves displays matching the provided filter
func (r *Repository) List(ctx context.Context, filter display.DisplayFilter) ([]*display.Display, error) {
	const op = "DisplayRepository.List"

	query := `
		SELECT 
			id, name, site_id, zone, position,
			state, last_seen, version, properties
		FROM displays
		WHERE 1=1
	`
	var args []interface{}
	var conditions []string

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

	for _, cond := range conditions {
		query += " AND " + cond
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to list displays",
			"error", err,
			"filter", filter,
			"operation", op,
		)
		return nil, database.MapError(err, op)
	}
	defer rows.Close()

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
			r.logger.Error("failed to scan display row",
				"error", err,
				"operation", op,
			)
			return nil, database.MapError(err, op)
		}

		if err := json.Unmarshal(propertiesJSON, &d.Properties); err != nil {
			r.logger.Error("failed to unmarshal properties",
				"error", err,
				"displayID", d.ID,
				"operation", op,
			)
			return nil, fmt.Errorf("error unmarshaling properties: %w", err)
		}

		displays = append(displays, &d)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating display rows",
			"error", err,
			"operation", op,
		)
		return nil, database.MapError(err, op)
	}

	r.logger.Info("listed displays",
		"count", len(displays),
		"filter", filter,
		"operation", op,
	)
	return displays, nil
}

// Delete removes a display from storage
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	const op = "DisplayRepository.Delete"

	result, err := r.db.ExecContext(ctx, `
		DELETE FROM displays
		WHERE id = $1
	`, id)
	if err != nil {
		r.logger.Error("failed to delete display",
			"error", err,
			"displayID", id,
			"operation", op,
		)
		return database.MapError(err, op)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("failed to get rows affected",
			"error", err,
			"displayID", id,
			"operation", op,
		)
		return database.MapError(err, op)
	}

	if rows == 0 {
		r.logger.Warn("display not found during delete",
			"displayID", id,
			"operation", op,
		)
		return database.MapError(sql.ErrNoRows, op)
	}

	r.logger.Info("deleted display",
		"displayID", id,
		"operation", op,
	)
	return nil
}
