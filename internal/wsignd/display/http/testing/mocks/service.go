package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
)

// Service implements a mock display service
type Service struct {
	mock.Mock
}

func (m *Service) Register(ctx context.Context, name string, location display.Location) (*display.Display, error) {
	args := m.Called(ctx, name, location)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*display.Display), args.Error(1)
}

func (m *Service) Get(ctx context.Context, id uuid.UUID) (*display.Display, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*display.Display), args.Error(1)
}

func (m *Service) GetByName(ctx context.Context, name string) (*display.Display, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*display.Display), args.Error(1)
}

func (m *Service) List(ctx context.Context, filter display.DisplayFilter) ([]*display.Display, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*display.Display), args.Error(1)
}

func (m *Service) UpdateLocation(ctx context.Context, id uuid.UUID, location display.Location) error {
	args := m.Called(ctx, id, location)
	return args.Error(0)
}

func (m *Service) Activate(ctx context.Context, id uuid.UUID) (*display.Display, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*display.Display), args.Error(1)
}

func (m *Service) Disable(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *Service) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *Service) SetProperty(ctx context.Context, id uuid.UUID, key, value string) error {
	args := m.Called(ctx, id, key, value)
	return args.Error(0)
}
