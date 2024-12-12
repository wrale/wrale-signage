package postgres

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/google/uuid"
	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	"github.com/wrale/wrale-signage/internal/wsignd/database"
	werrors "github.com/wrale/wrale-signage/internal/wsignd/errors"
)

type repository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewRepository creates a new PostgreSQL token repository
func NewRepository(db *sql.DB, logger *slog.Logger) auth.Repository {
	return &repository{
		db:     db,
		logger: logger,
	}
}

func (r *repository) Save(ctx context.Context, token *auth.Token) error {
	const op = "TokenRepository.Save"

	err := database.RunInTx(ctx, r.db, nil, func(tx *database.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO access_tokens (
				id, display_id,
				access_token_hash, refresh_token_hash,
				access_token_expires_at, refresh_token_expires_at,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			token.ID,
			token.DisplayID,
			token.AccessTokenHash,
			token.RefreshTokenHash,
			token.AccessTokenExpiry,
			token.RefreshTokenExpiry,
			token.CreatedAt,
			token.UpdatedAt,
		)
		return err
	})

	if err != nil {
		return werrors.NewError("DB_ERROR", "failed to save token", op, err)
	}
	return nil
}

func (r *repository) FindByDisplayID(ctx context.Context, displayID uuid.UUID) (*auth.Token, error) {
	const op = "TokenRepository.FindByDisplayID"

	var token auth.Token
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, display_id,
			access_token_hash, refresh_token_hash,
			access_token_expires_at, refresh_token_expires_at,
			created_at, updated_at
		FROM access_tokens
		WHERE display_id = $1
		AND access_token_expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, displayID).Scan(
		&token.ID,
		&token.DisplayID,
		&token.AccessTokenHash,
		&token.RefreshTokenHash,
		&token.AccessTokenExpiry,
		&token.RefreshTokenExpiry,
		&token.CreatedAt,
		&token.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, auth.ErrTokenNotFound
	}
	if err != nil {
		return nil, werrors.NewError("DB_ERROR", "failed to find token", op, err)
	}

	return &token, nil
}

func (r *repository) FindByAccessToken(ctx context.Context, tokenHash []byte) (*auth.Token, error) {
	const op = "TokenRepository.FindByAccessToken"

	var token auth.Token
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, display_id,
			access_token_hash, refresh_token_hash,
			access_token_expires_at, refresh_token_expires_at,
			created_at, updated_at
		FROM access_tokens
		WHERE access_token_hash = $1
		AND access_token_expires_at > NOW()
	`, tokenHash).Scan(
		&token.ID,
		&token.DisplayID,
		&token.AccessTokenHash,
		&token.RefreshTokenHash,
		&token.AccessTokenExpiry,
		&token.RefreshTokenExpiry,
		&token.CreatedAt,
		&token.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, auth.ErrTokenNotFound
	}
	if err != nil {
		return nil, werrors.NewError("DB_ERROR", "failed to find token", op, err)
	}

	return &token, nil
}

func (r *repository) FindByRefreshToken(ctx context.Context, tokenHash []byte) (*auth.Token, error) {
	const op = "TokenRepository.FindByRefreshToken"

	var token auth.Token
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			id, display_id,
			access_token_hash, refresh_token_hash,
			access_token_expires_at, refresh_token_expires_at,
			created_at, updated_at
		FROM access_tokens
		WHERE refresh_token_hash = $1
		AND refresh_token_expires_at > NOW()
	`, tokenHash).Scan(
		&token.ID,
		&token.DisplayID,
		&token.AccessTokenHash,
		&token.RefreshTokenHash,
		&token.AccessTokenExpiry,
		&token.RefreshTokenExpiry,
		&token.CreatedAt,
		&token.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, auth.ErrTokenNotFound
	}
	if err != nil {
		return nil, werrors.NewError("DB_ERROR", "failed to find token", op, err)
	}

	return &token, nil
}

func (r *repository) DeleteByDisplayID(ctx context.Context, displayID uuid.UUID) error {
	const op = "TokenRepository.DeleteByDisplayID"

	_, err := r.db.ExecContext(ctx, `
		DELETE FROM access_tokens
		WHERE display_id = $1
	`, displayID)

	if err != nil {
		return werrors.NewError("DB_ERROR", "failed to delete tokens", op, err)
	}
	return nil
}
