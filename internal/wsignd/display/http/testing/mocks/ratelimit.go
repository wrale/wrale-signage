package mocks

import (
	"context"
	"fmt"
	"time"

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
	limit, ok := args.Get(0).(ratelimit.Limit)
	if !ok {
		panic(fmt.Sprintf("GetLimit: invalid return type for limit type %q", limitType))
	}
	return limit
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

// Status returns the current rate limit status for a key.
// It safely handles both success and error cases, with proper nil checking
// and type assertions.
func (m *RateLimitService) Status(key ratelimit.LimitKey) (*ratelimit.LimitStatus, error) {
	args := m.Called(key)

	// For error cases, first argument should be nil
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	// For success cases, safely convert to LimitStatus
	status, ok := args.Get(0).(*ratelimit.LimitStatus)
	if !ok {
		panic(fmt.Sprintf(
			"Status: invalid return type for rate limit status (key=%+v). Expected *ratelimit.LimitStatus, got %T",
			key, args.Get(0),
		))
	}

	return status, args.Error(1)
}

// DefaultStatus creates a default success status for testing.
// It properly calculates the reset time as current time plus the rate limit period,
// matching the behavior of the real implementation.
func DefaultStatus(limit ratelimit.Limit, remaining int) *ratelimit.LimitStatus {
	return &ratelimit.LimitStatus{
		Limit:     limit,
		Remaining: remaining,
		Reset:     time.Now().Add(limit.Period), // Calculate actual reset time
		Period:    limit.Period,
	}
}
