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

// RegisterConfiguredLimits sets up rate limits from configuration
func (s *RateLimitService) RegisterConfiguredLimits(cfg config.RateLimitConfig) {
	var errs []error

	// Token rate limits
	if err := s.RegisterLimit("token_refresh", Limit{
		Rate:        cfg.API.TokenRefreshPerHour,
		Period:      time.Hour,
		BurstSize:   cfg.API.RefreshBurstSize,
		WaitTimeout: 0, // No waiting for tokens
	}); err != nil {
		errs = append(errs, fmt.Errorf("token_refresh: %w", err))
	}

	// API rate limits
	if err := s.RegisterLimit("api_request", Limit{
		Rate:        cfg.API.RequestsPerMinute,
		Period:      time.Minute,
		BurstSize:   cfg.API.BurstSize,
		WaitTimeout: cfg.Settings.MaxWaitTime,
	}); err != nil {
		errs = append(errs, fmt.Errorf("api_request: %w", err))
	}

	// Device code limits
	if err := s.RegisterLimit("device_code", Limit{
		Rate:        cfg.API.DeviceCodePerInterval,
		Period:      cfg.API.DeviceCodeInterval,
		BurstSize:   0, // No bursts for security
		WaitTimeout: 0, // No waiting
	}); err != nil {
		errs = append(errs, fmt.Errorf("device_code: %w", err))
	}

	// WebSocket limits
	if err := s.RegisterLimit("ws_connection", Limit{
		Rate:        cfg.WebSocket.MessagesPerMinute,
		Period:      time.Minute,
		BurstSize:   cfg.WebSocket.BurstSize,
		WaitTimeout: cfg.Settings.MaxWaitTime,
	}); err != nil {
		errs = append(errs, fmt.Errorf("ws_connection: %w", err))
	}

	if len(errs) > 0 {
		s.logger.Error("failed to register rate limits",
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

// RegisterDefaultLimits configures standard rate limits
func (s *RateLimitService) RegisterDefaultLimits() {
	var errs []error

	// Token rate limits
	if err := s.RegisterLimit("token_refresh", Limit{
		Rate:        5, // 5 refreshes
		Period:      time.Hour,
		BurstSize:   2, // Allow small burst
		WaitTimeout: 0, // No waiting
	}); err != nil {
		errs = append(errs, fmt.Errorf("token_refresh: %w", err))
	}

	// API rate limits
	if err := s.RegisterLimit("api_request", Limit{
		Rate:        120, // 120 requests
		Period:      time.Minute,
		BurstSize:   30,          // Allow bursts
		WaitTimeout: time.Second, // Short wait allowed
	}); err != nil {
		errs = append(errs, fmt.Errorf("api_request: %w", err))
	}

	// WebSocket limits
	if err := s.RegisterLimit("ws_message", Limit{
		Rate:        60, // 60 messages
		Period:      time.Minute,
		BurstSize:   15, // Allow message bursts
		WaitTimeout: 0,  // No waiting for WS
	}); err != nil {
		errs = append(errs, fmt.Errorf("ws_message: %w", err))
	}

	// Registration limits
	if err := s.RegisterLimit("device_code", Limit{
		Rate:        10,          // 10 attempts (increased from 3)
		Period:      time.Minute, // 1 minute (reduced from 5)
		BurstSize:   0,           // No bursts
		WaitTimeout: 0,           // No waiting
	}); err != nil {
		errs = append(errs, fmt.Errorf("device_code: %w", err))
	}

	if len(errs) > 0 {
		s.logger.Error("failed to register default rate limits",
			"errors", errs,
		)
	}
}

// BulkReset clears rate limits for multiple keys atomically
func (s *RateLimitService) BulkReset(ctx context.Context, keys []LimitKey) error {
	for _, key := range keys {
		if key.Type == "" {
			return fmt.Errorf("%w: empty type in key", ErrInvalidKey)
		}
	}

	for _, key := range keys {
		if err := s.store.Reset(ctx, key); err != nil {
			s.logger.Error("failed to reset rate limit in bulk operation",
				"error", err,
				"type", key.Type,
				"token", key.Token,
				"endpoint", key.Endpoint,
			)
			return err
		}
	}

	return nil
}
