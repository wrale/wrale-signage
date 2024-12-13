package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/config"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// RateLimitService implements a mock rate limiter
type RateLimitService struct {
	mock.Mock
}

func (m *RateLimitService) Allow(ctx context.Context, key ratelimit.LimitKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *RateLimitService) GetLimit(limitType string) ratelimit.Limit {
	args := m.Called(limitType)
	return args.Get(0).(ratelimit.Limit)
}

func (m *RateLimitService) Reset(ctx context.Context, key ratelimit.LimitKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *RateLimitService) RegisterDefaultLimits() {
	m.Called()
}

func (m *RateLimitService) RegisterConfiguredLimits(cfg config.RateLimitConfig) {
	m.Called(cfg)
}
