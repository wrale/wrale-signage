package postgres

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
)

type Repository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewRepository(db *sql.DB, logger *slog.Logger) activation.Repository {
	return &Repository{db: db, logger: logger}
}

func (r *Repository) Save(code *activation.DeviceCode) error {
	const op = "DeviceCodeRepository.Save"

	return database.RunInTx(context.Background(), r.db, nil, func(tx *database.Tx) error {
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO device_codes (
				id, device_code, user_code,
				expires_at, poll_interval, activated,
				activated_at, display_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				activated = EXCLUDED.activated,
				activated_at = EXCLUDED.activated_at,
				display_id = EXCLUDED.display_id
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
}

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

	if err == sql.ErrNoRows {
		return nil, activation.ErrCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &dc, nil
}

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

	if err == sql.ErrNoRows {
		return nil, activation.ErrCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &dc, nil
}

func (r *Repository) Delete(id uuid.UUID) error {
	const op = "DeviceCodeRepository.Delete"

	_, err := r.db.ExecContext(context.Background(), `
		DELETE FROM device_codes
		WHERE id = $1
	`, id)
	return err
}
