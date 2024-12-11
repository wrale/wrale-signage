package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
)

// Repository implements activation.Repository using PostgreSQL
type Repository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewRepository creates a new PostgreSQL device code repository
func NewRepository(db *sql.DB, logger *slog.Logger) activation.Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// Save persists a device code to the database
func (r *Repository) Save(code *activation.DeviceCode) error {
	const op = "DeviceCodeRepository.Save"

	err := database.RunInTx(context.Background(), r.db, nil, func(tx *database.Tx) error {
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO device_codes (
				id, device_code, user_code,
				expires_at, poll_interval, activated,
				activated_at, display_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			code.ID,
			code.DeviceCode,
			code.UserCode,
			code.ExpiresAt,
			code.PollInterval,
			code.Activated,
			code.ActivatedAt,
			code.DisplayID,
		)
		return err
	})

	if err != nil {
		r.logger.Error("failed to save device code",
			"error", err,
			"deviceCodeID", code.ID,
			"operation", op,
		)
		return database.MapError(err, op)
	}

	return nil
}

// FindByDeviceCode retrieves a device code by its opaque device code
func (r *Repository) FindByDeviceCode(code string) (*activation.DeviceCode, error) {
	const op = "DeviceCodeRepository.FindByDeviceCode"

	var dc activation.DeviceCode
	err := r.db.QueryRowContext(context.Background(), `
		SELECT 
			id, device_code, user_code,
			expires_at, poll_interval, activated,
			activated_at, display_id
		FROM device_codes
		WHERE device_code = $1
	`, code).Scan(
		&dc.ID,
		&dc.DeviceCode,
		&dc.UserCode,
		&dc.ExpiresAt,
		&dc.PollInterval,
		&dc.Activated,
		&dc.ActivatedAt,
		&dc.DisplayID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Warn("device code not found",
				"deviceCode", code,
				"operation", op,
			)
		} else {
			r.logger.Error("failed to find device code",
				"error", err,
				"deviceCode", code,
				"operation", op,
			)
		}
		return nil, database.MapError(err, op)
	}

	return &dc, nil
}

// FindByUserCode retrieves a device code by its human-readable user code
func (r *Repository) FindByUserCode(code string) (*activation.DeviceCode, error) {
	const op = "DeviceCodeRepository.FindByUserCode"

	var dc activation.DeviceCode
	err := r.db.QueryRowContext(context.Background(), `
		SELECT 
			id, device_code, user_code,
			expires_at, poll_interval, activated,
			activated_at, display_id
		FROM device_codes
		WHERE user_code = $1
	`, code).Scan(
		&dc.ID,
		&dc.DeviceCode,
		&dc.UserCode,
		&dc.ExpiresAt,
		&dc.PollInterval,
		&dc.Activated,
		&dc.ActivatedAt,
		&dc.DisplayID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Warn("device code not found",
				"userCode", code,
				"operation", op,
			)
		} else {
			r.logger.Error("failed to find device code",
				"error", err,
				"userCode", code,
				"operation", op,
			)
		}
		return nil, database.MapError(err, op)
	}

	return &dc, nil
}

// Delete removes a device code
func (r *Repository) Delete(id uuid.UUID) error {
	const op = "DeviceCodeRepository.Delete"

	result, err := r.db.ExecContext(context.Background(), `
		DELETE FROM device_codes
		WHERE id = $1
	`, id)
	if err != nil {
		r.logger.Error("failed to delete device code",
			"error", err,
			"deviceCodeID", id,
			"operation", op,
		)
		return database.MapError(err, op)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("failed to get rows affected",
			"error", err,
			"deviceCodeID", id,
			"operation", op,
		)
		return database.MapError(err, op)
	}

	if rows == 0 {
		r.logger.Warn("device code not found during delete",
			"deviceCodeID", id,
			"operation", op,
		)
		return fmt.Errorf("device code not found")
	}

	return nil
}
