package testing

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	displayhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http"
	"github.com/wrale/wrale-signage/internal/wsignd/display/http/testing/mocks"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// Common rate limit types from handler configuration
const (
	apiRequestLimit   = "api_request"
	deviceCodeLimit   = "device_code"
	tokenRefreshLimit = "token_refresh"
	wsConnectionLimit = "ws_connection"
)

// TestHandler provides access to handler and mocks for testing
type TestHandler struct {
	Handler    *displayhttp.Handler
	Service    *mocks.Service
	Activation *mocks.ActivationService
	Auth       *mocks.AuthService
	RateLimit  *mocks.RateLimitService
	logger     *slog.Logger
}

// NewTestHandler creates a new handler with mock services for testing
func NewTestHandler() *TestHandler {
	// Create mocks
	mockSvc := &mocks.Service{}
	mockActSvc := &mocks.ActivationService{}
	mockAuthSvc := &mocks.AuthService{}
	mockRateLimitSvc := &mocks.RateLimitService{}

	// Create logger that writes to stdout for test visibility
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create handler with mocks
	handler := displayhttp.NewHandler(mockSvc, mockActSvc, mockAuthSvc, mockRateLimitSvc, logger)

	return &TestHandler{
		Handler:    handler,
		Service:    mockSvc,
		Activation: mockActSvc,
		Auth:       mockAuthSvc,
		RateLimit:  mockRateLimitSvc,
		logger:     logger,
	}
}

// MockRequest creates a test request with proper test context
func (th *TestHandler) MockRequest(method, target string, body interface{}) (*http.Request, error) {
	var bodyReader *httptest.ResponseRecorder
	if body != nil {
		bodyReader = httptest.NewRecorder()
		if err := json.NewEncoder(bodyReader).Encode(body); err != nil {
			return nil, err
		}
	}

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, target, bodyReader.Body)
	} else {
		req, err = http.NewRequest(method, target, nil)
	}
	if err != nil {
		return nil, err
	}

	// Add required headers
	req.Header.Set("X-Real-IP", "192.0.2.1:1234")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add chi routing context
	chiCtx := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	// Add request ID
	req = req.WithContext(context.WithValue(req.Context(),
		middleware.RequestIDKey, "test-request-id"))

	return req, nil
}

// MockAuthorizedRequest creates a test request with auth token
func (th *TestHandler) MockAuthorizedRequest(method, target string, body interface{}) (*http.Request, error) {
	req, err := th.MockRequest(method, target, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer test-token")
	return req, nil
}

// WriteJSON is a test helper that matches the production writeJSON behavior
func WriteJSON(w http.ResponseWriter, status int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// SetupRateLimitBypass configures rate limit mock to allow all requests
func (th *TestHandler) SetupRateLimitBypass() {
	// Default rate limit configuration
	limit := ratelimit.Limit{
		Rate:      100,
		Period:    time.Minute,
		BurstSize: 10,
	}

	// Set up rate limit lookups
	limitTypes := []string{
		apiRequestLimit,
		deviceCodeLimit,
		tokenRefreshLimit,
		wsConnectionLimit,
	}

	for _, limitType := range limitTypes {
		th.RateLimit.On("GetLimit", limitType).Return(limit)
	}

	// Set up allow checks with proper rate limit keys
	th.RateLimit.On("Allow", mock.Anything, mock.MatchedBy(func(key ratelimit.LimitKey) bool {
		// Allow any rate limit key that has a valid type
		return key.Type != ""
	})).Return(nil)
}

// ValidateJSON attempts to decode JSON into a map and verify its structure
func (th *TestHandler) ValidateJSON(body []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CleanupTest performs cleanup after each test
func (th *TestHandler) CleanupTest() {
	th.Service.AssertExpectations(th.t)
	th.Activation.AssertExpectations(th.t)
	th.Auth.AssertExpectations(th.t)
	th.RateLimit.AssertExpectations(th.t)
}
