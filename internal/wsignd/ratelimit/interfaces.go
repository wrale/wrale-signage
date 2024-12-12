package ratelimit

import (
	"context"
	"time"

	"github.com/wrale/wrale-signage/internal/wsignd/config"
)

// LimitKey identifies a specific rate limit
type LimitKey struct {
	Type     string // e.g., "refresh", "ws_message"
	Token    string // auth token or other unique identifier
	RemoteIP string // remote IP for unauthenticated limits
	Endpoint string // API endpoint for specific limits
}

// Store handles rate limit state persistence
type Store interface {
	// Increment attempts to increment a counter and returns the current count
	// Returns error if limit is exceeded
	Increment(ctx context.Context, key LimitKey, limit Limit) (int, error)

	// Reset clears a rate limit counter
	Reset(ctx context.Context, key LimitKey) error

	// IsExceeded checks if a limit has been exceeded without incrementing
	IsExceeded(ctx context.Context, key LimitKey, limit Limit) (bool, error)
}

// Service manages rate limiting for the application
type Service interface {
	// Allow checks if an operation should be allowed
	Allow(ctx context.Context, key LimitKey) error

	// GetLimit returns the configured limit for a key type
	GetLimit(limitType string) Limit

	// Reset clears rate limit counters for a key
	Reset(ctx context.Context, key LimitKey) error

	// RegisterDefaultLimits configures standard rate limits
	RegisterDefaultLimits()

	// RegisterConfiguredLimits configures rate limits from config
	RegisterConfiguredLimits(config.RateLimitConfig)
}

// Limit defines the rate limit configuration
type Limit struct {
	// Rate is the number of operations allowed
	Rate int

	// Period is the time window for the rate
	Period time.Duration

	// BurstSize allows a short burst over the rate (optional)
	BurstSize int

	// WaitTimeout is how long to wait if rate limited (0 for no wait)
	WaitTimeout time.Duration
}

// Error types for rate limiting
var (
	ErrLimitExceeded = NewError("RATE_LIMITED", "rate limit exceeded")
	ErrStoreError    = NewError("STORE_ERROR", "rate limit store error")
	ErrInvalidLimit  = NewError("INVALID_LIMIT", "invalid rate limit configuration")
	ErrInvalidKey    = NewError("INVALID_KEY", "invalid rate limit key")
)

// Error represents a rate limiting error
type Error struct {
	Code    string
	Message string
}

func (e Error) Error() string {
	return e.Message
}

// NewError creates a new rate limit error
func NewError(code string, message string) Error {
	return Error{
		Code:    code,
		Message: message,
	}
}
