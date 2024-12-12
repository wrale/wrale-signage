package auth

import (
	"context"
	"crypto/sha256"
	"log/slog"

	"github.com/google/uuid"
)

// Service manages display authentication
type Service interface {
	// CreateToken generates a new token pair for a display
	CreateToken(ctx context.Context, displayID uuid.UUID) (*Token, error)

	// ValidateAccessToken validates an access token and returns the associated display ID
	ValidateAccessToken(ctx context.Context, token string) (uuid.UUID, error)

	// RefreshToken generates a new token pair using a refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*Token, error)

	// RevokeTokens invalidates all tokens for a display
	RevokeTokens(ctx context.Context, displayID uuid.UUID) error
}

// Repository defines storage operations for tokens
type Repository interface {
	// Save stores a token
	Save(ctx context.Context, token *Token) error

	// FindByDisplayID gets the current token for a display
	FindByDisplayID(ctx context.Context, displayID uuid.UUID) (*Token, error)

	// FindByAccessToken finds a token by its access token hash
	FindByAccessToken(ctx context.Context, tokenHash []byte) (*Token, error)

	// FindByRefreshToken finds a token by its refresh token hash
	FindByRefreshToken(ctx context.Context, tokenHash []byte) (*Token, error)

	// DeleteByDisplayID removes all tokens for a display
	DeleteByDisplayID(ctx context.Context, displayID uuid.UUID) error
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService creates a new authentication service
func NewService(repo Repository, logger *slog.Logger) Service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) CreateToken(ctx context.Context, displayID uuid.UUID) (*Token, error) {
	// First revoke any existing tokens
	if err := s.repo.DeleteByDisplayID(ctx, displayID); err != nil {
		s.logger.Error("failed to revoke existing tokens",
			"error", err,
			"displayID", displayID,
		)
		return nil, err
	}

	// Generate new token pair
	token, err := NewToken(displayID)
	if err != nil {
		s.logger.Error("failed to generate token",
			"error", err,
			"displayID", displayID,
		)
		return nil, err
	}

	// Save to repository
	if err := s.repo.Save(ctx, token); err != nil {
		s.logger.Error("failed to save token",
			"error", err,
			"displayID", displayID,
		)
		return nil, err
	}

	return token, nil
}

func (s *service) ValidateAccessToken(ctx context.Context, token string) (uuid.UUID, error) {
	// Hash token for lookup
	hash := sha256.Sum256([]byte(token))

	// Find token by hash
	storedToken, err := s.repo.FindByAccessToken(ctx, hash[:])
	if err != nil {
		s.logger.Error("failed to find token",
			"error", err,
		)
		return uuid.Nil, err
	}

	// Validate token
	if err := storedToken.ValidateAccessToken(token); err != nil {
		s.logger.Error("token validation failed",
			"error", err,
			"displayID", storedToken.DisplayID,
		)
		return uuid.Nil, err
	}

	return storedToken.DisplayID, nil
}

func (s *service) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	// Hash token for lookup
	hash := sha256.Sum256([]byte(refreshToken))

	// Find token by refresh token hash
	storedToken, err := s.repo.FindByRefreshToken(ctx, hash[:])
	if err != nil {
		s.logger.Error("failed to find token for refresh",
			"error", err,
		)
		return nil, err
	}

	// Validate refresh token
	if err := storedToken.ValidateRefreshToken(refreshToken); err != nil {
		s.logger.Error("refresh token validation failed",
			"error", err,
			"displayID", storedToken.DisplayID,
		)
		return nil, err
	}

	// Generate new token pair
	return s.CreateToken(ctx, storedToken.DisplayID)
}

func (s *service) RevokeTokens(ctx context.Context, displayID uuid.UUID) error {
	if err := s.repo.DeleteByDisplayID(ctx, displayID); err != nil {
		s.logger.Error("failed to revoke tokens",
			"error", err,
			"displayID", displayID,
		)
		return err
	}
	return nil
}
