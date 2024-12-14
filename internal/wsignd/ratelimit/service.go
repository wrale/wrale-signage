package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/wrale/wrale-signage/internal/wsignd/config"
)

// RateLimitService implements the Service interface
type RateLimitService struct {
	store   Store
	logger  *slog.Logger
	limits  map[string]Limit
	limitsM sync.RWMutex
}

// NewService creates a new rate limiting service
func NewService(store Store, logger *slog.Logger) Service {
	return &RateLimitService{
		store:  store,
		logger: logger,
		limits: make(map[string]Limit),
	}
}

// Status returns the current remaining requests and reset time for a key
func (s *RateLimitService) Status(key LimitKey) (remaining int, reset time.Time, err error) {
	if key.Type == "" {
		return 0, time.Time{}, ErrInvalidKey
	}

	// Get the configured limit
	limit := s.GetLimit(key.Type)
	if limit.Rate == 0 {
		// If no limit configured, return "infinite" remaining
		return -1, time.Now().Add(limit.Period), nil
	}

	// Get current count from store
	count, err := s.store.GetCount(context.Background(), key)
	if err != nil {
		s.logger.Error("failed to get rate limit count",
			"error", err,
			"type", key.Type,
			"token", key.Token,
			"endpoint", key.Endpoint,
		)
		return 0, time.Time{}, err
	}

	// Calculate remaining and next reset time
	remaining = limit.Rate - count
	if remaining < 0 {
		remaining = 0
	}

	// Calculate when this window resets
	// Note: This is an approximation based on the Period
	reset = time.Now().Add(limit.Period)

	return remaining, reset, nil
}

// Allow checks if an operation should be allowed
func (s *RateLimitService) Allow(ctx context.Context, key LimitKey) error {
	if key.Type == "" {
		return ErrInvalidKey
	}

	limit := s.GetLimit(key.Type)
	if limit.Rate == 0 {
		s.logger.Warn("no rate limit configured for type",
			"type", key.Type,
		)
		// Allow if no limit configured
		return nil
	}

	count, err := s.store.Increment(ctx, key, limit)
	if err != nil {
		s.logger.Error("rate limit check failed",
			"error", err,
			"type", key.Type,
			"token", key.Token,
			"endpoint", key.Endpoint,
		)
		return err
	}

	s.logger.Debug("rate limit check",
		"type", key.Type,
		"count", count,
		"limit", limit.Rate,
		"burst", limit.BurstSize,
		"token", key.Token,
		"endpoint", key.Endpoint,
	)

	return nil
}

// GetLimit returns the configured limit for a key type
func (s *RateLimitService) GetLimit(limitType string) Limit {
	s.limitsM.RLock()
	defer s.limitsM.RUnlock()

	return s.limits[limitType]
}

// Reset clears rate limit counters for a key
func (s *RateLimitService) Reset(ctx context.Context, key LimitKey) error {
	if key.Type == "" {
		return ErrInvalidKey
	}

	if err := s.store.Reset(ctx, key); err != nil {
		s.logger.Error("failed to reset rate limit",
			"error", err,
			"type", key.Type,
			"token", key.Token,
			"endpoint", key.Endpoint,
		)
		return err
	}

	return nil
}

// RegisterConfiguredLimits sets up rate limits from configuration
func (s *RateLimitService) RegisterConfiguredLimits(cfg config.RateLimitConfig) {
	// Register each configured limit, with built-in defaults
	limits := []struct {
		name  string
		limit Limit
	}{
		{
			name: "token_refresh",
			limit: Limit{
				Rate:        cfg.API.TokenRefreshPerHour,
				Period:      time.Hour,
				BurstSize:   cfg.API.RefreshBurstSize,
				WaitTimeout: 0,
			},
		},
		{
			name: "api_request",
			limit: Limit{
				Rate:        cfg.API.RequestsPerMinute,
				Period:      time.Minute,
				BurstSize:   cfg.API.BurstSize,
				WaitTimeout: cfg.Settings.MaxWaitTime,
			},
		},
		{
			name: "device_code",
			limit: Limit{
				Rate:        cfg.API.DeviceCodePerInterval,
				Period:      cfg.API.DeviceCodeInterval,
				BurstSize:   0,
				WaitTimeout: 0,
			},
		},
		{
			name: "ws_connection",
			limit: Limit{
				Rate:        cfg.WebSocket.MessagesPerMinute,
				Period:      time.Minute,
				BurstSize:   cfg.WebSocket.BurstSize,
				WaitTimeout: cfg.Settings.MaxWaitTime,
			},
		},
	}

	var errs []error
	for _, l := range limits {
		if err := s.RegisterLimit(l.name, l.limit); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", l.name, err))
		}
	}

	if len(errs) > 0 {
		s.logger.Error("failed to register rate limits",
			"errors", errs,
		)
	}
}

// RegisterDefaultLimits configures standard rate limits
func (s *RateLimitService) RegisterDefaultLimits() {
	// Define sensible defaults that balance security with usability
	limits := []struct {
		name  string
		limit Limit
	}{
		{
			name: "token_refresh",
			limit: Limit{
				Rate:        5, // 5 refreshes per hour
				Period:      time.Hour,
				BurstSize:   2, // Small burst allowed
				WaitTimeout: 0, // No waiting for tokens
			},
		},
		{
			name: "api_request",
			limit: Limit{
				Rate:        120, // 120 requests per minute
				Period:      time.Minute,
				BurstSize:   30,          // Allow substantial bursts
				WaitTimeout: time.Second, // Short wait allowed
			},
		},
		{
			name: "device_code",
			limit: Limit{
				Rate:        10, // 10 attempts per minute
				Period:      time.Minute,
				BurstSize:   0, // No bursts for security
				WaitTimeout: 0, // No waiting
			},
		},
		{
			name: "ws_connection",
			limit: Limit{
				Rate:        60, // 60 messages per minute
				Period:      time.Minute,
				BurstSize:   15, // Allow message bursts
				WaitTimeout: 0,  // No waiting for WS
			},
		},
	}

	var errs []error
	for _, l := range limits {
		if err := s.RegisterLimit(l.name, l.limit); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", l.name, err))
		}
	}

	if len(errs) > 0 {
		s.logger.Error("failed to register default rate limits",
			"errors", errs,
		)
	}
}

// RegisterLimit adds or updates a rate limit configuration
func (s *RateLimitService) RegisterLimit(limitType string, limit Limit) error {
	if limit.Rate <= 0 || limit.Period <= 0 {
		return ErrInvalidLimit
	}

	s.limitsM.Lock()
	defer s.limitsM.Unlock()

	s.limits[limitType] = limit
	return nil
}
