package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/auth"
)

// AuthService implements a mock auth service
type AuthService struct {
	mock.Mock
}

func (m *AuthService) CreateToken(ctx context.Context, displayID uuid.UUID) (*auth.Token, error) {
	args := m.Called(ctx, displayID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Token), args.Error(1)
}

func (m *AuthService) ValidateAccessToken(ctx context.Context, token string) (uuid.UUID, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.Token, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.Token), args.Error(1)
}

func (m *AuthService) RevokeTokens(ctx context.Context, displayID uuid.UUID) error {
	args := m.Called(ctx, displayID)
	return args.Error(0)
}
