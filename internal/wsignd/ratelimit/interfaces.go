package ratelimit

import (
	"context"
	"net/http"
	"time"

	"github.com/wrale/wrale-signage/internal/wsignd/config"
)

// LimitKey identifies a specific rate limit. It combines all the information
// needed to uniquely identify a rate limit counter.
type LimitKey struct {
	Type     string // e.g., "refresh", "ws_message"
	Token    string // auth token or other unique identifier
	RemoteIP string // remote IP for unauthenticated limits
	Endpoint string // API endpoint for specific limits
}

// LimitStatus represents the complete state of a rate limit at a point in time.
// This provides all the information needed for rate limit headers and decisions.
type LimitStatus struct {
	// Remaining is the number of requests remaining in the current window
	Remaining int

	// Reset is when the current window expires and the limit resets
	Reset time.Time

	// Limit is the configured limit that applies
	Limit Limit

	// Period is the current time window
	Period time.Duration
}

// Store handles rate limit state persistence. It provides the low-level
// operations needed to track and enforce rate limits.
type Store interface {
	// Increment attempts to increment a counter and returns the current count.
	// Returns error if limit is exceeded.
	Increment(ctx context.Context, key LimitKey, limit Limit) (int, error)

	// Reset clears a rate limit counter, typically used when limits change
	// or for administrative purposes.
	Reset(ctx context.Context, key LimitKey) error

	// IsExceeded checks if a limit has been exceeded without incrementing.
	// This allows for passive checking without affecting the counter.
	IsExceeded(ctx context.Context, key LimitKey, limit Limit) (bool, error)

	// GetCount returns the current count for a key without any side effects.
	GetCount(ctx context.Context, key LimitKey) (int, error)
}

// Service manages rate limiting for the application. It provides the high-level
// interface for rate limit enforcement and configuration.
type Service interface {
	// Allow checks if an operation should be allowed
	Allow(ctx context.Context, key LimitKey) error

	// GetLimit returns the configured limit for a key type
	GetLimit(limitType string) Limit

	// Status returns the current rate limit status for a key
	Status(key LimitKey) (*LimitStatus, error)

	// Reset clears rate limit counters for a key
	Reset(ctx context.Context, key LimitKey) error

	// RegisterDefaultLimits configures standard rate limits
	RegisterDefaultLimits()

	// RegisterConfiguredLimits configures rate limits from config
	RegisterConfiguredLimits(config.RateLimitConfig)
}

// Limit defines the rate limit configuration. It contains all parameters
// that control how the rate limit behaves.
type Limit struct {
	// Rate is the number of operations allowed in the Period
	Rate int

	// Period is the time window for the rate (e.g., 1 hour, 1 minute)
	Period time.Duration

	// BurstSize allows a short burst over the rate (optional)
	BurstSize int

	// WaitTimeout is how long to wait if rate limited (0 for no wait)
	WaitTimeout time.Duration
}

// RateLimitOptions configures the behavior of rate limiting middleware.
// It combines configuration from both the original interfaces.go and middleware.go
// to provide all needed functionality in one place.
type RateLimitOptions struct {
	// LimitType identifies which rate limit to apply (required)
	LimitType string

	// GetToken extracts an authentication token from the request.
	// This is used for per-token rate limiting. If not provided,
	// rate limits are applied based on IP address only.
	GetToken func(r *http.Request) string

	// WaitOnLimit indicates if requests should wait when rate limited.
	// If true, the middleware will retry with backoff up to WaitTimeout.
	WaitOnLimit bool

	// WaitTimeout overrides the default wait timeout from the Limit.
	// This allows per-route customization of wait behavior.
	WaitTimeout time.Duration

	// SkipLimitCheck determines if rate limiting should be bypassed.
	// This is useful for health checks or admin endpoints that
	// should not be rate limited.
	SkipLimitCheck func(r *http.Request) bool
}

// Error types for rate limiting
var (
	// ErrLimitExceeded indicates the rate limit was exceeded
	ErrLimitExceeded = NewError("RATE_LIMITED", "rate limit exceeded")

	// ErrStoreError indicates a problem with the rate limit store
	ErrStoreError = NewError("STORE_ERROR", "rate limit store error")

	// ErrInvalidLimit indicates invalid rate limit configuration
	ErrInvalidLimit = NewError("INVALID_LIMIT", "invalid rate limit configuration")

	// ErrInvalidKey indicates a malformed or empty limit key
	ErrInvalidKey = NewError("INVALID_KEY", "invalid rate limit key")
)

// Error represents a rate limiting error with a machine-readable code
// and human-readable message.
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
