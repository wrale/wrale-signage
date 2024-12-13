package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
)

// ActivationService implements a mock activation service
type ActivationService struct {
	mock.Mock
}

func (m *ActivationService) GenerateCode(ctx context.Context) (*activation.DeviceCode, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*activation.DeviceCode), args.Error(1)
}

func (m *ActivationService) ActivateCode(ctx context.Context, code string, displayID uuid.UUID) error {
	args := m.Called(ctx, code, displayID)
	return args.Error(0)
}

func (m *ActivationService) ValidateCode(ctx context.Context, code string) (*activation.DeviceCode, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*activation.DeviceCode), args.Error(1)
}
