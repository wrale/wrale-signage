package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	displayhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http"
	"github.com/wrale/wrale-signage/internal/wsignd/display/http/testing/mocks"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

// Common rate limit types for test configuration
const (
	// Core rate limit types
	apiRequestLimit   = "api_request"   // General API request limiting
	deviceCodeLimit   = "device_code"   // Code generation/activation limits
	tokenRefreshLimit = "token_refresh" // Token refresh operation limits
	wsConnectionLimit = "ws_connection" // WebSocket connection limits

	// Test values
	testDisplayIDStr = "550e8400-e29b-41d4-a716-446655440000" // Test display UUID
	testToken        = "test-token"                           // Test auth token
)

// validRateLimitTypes maps known rate limit types to their descriptions
var validRateLimitTypes = map[string]string{
	apiRequestLimit:   "General API request limiting",
	deviceCodeLimit:   "Code generation/activation limits",
	tokenRefreshLimit: "Token refresh operation limits",
	wsConnectionLimit: "WebSocket connection limits",
}

// TestHandler provides access to handler and mocks for testing
type TestHandler struct {
	Handler    *displayhttp.Handler
	Service    *mocks.Service
	Activation *mocks.ActivationService
	Auth       *mocks.AuthService
	RateLimit  *mocks.RateLimitService
	logger     *slog.Logger
	t          *testing.T
}

// NewTestHandler creates a new handler with mock services for testing
func NewTestHandler(t *testing.T) *TestHandler {
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
		t:          t,
	}
}

// SetupTestDefaults configures common test prerequisites in the correct order
func (th *TestHandler) SetupTestDefaults() {
	// First set up auth bypass as it's most fundamental
	th.SetupAuthBypass()
	// Then set up rate limiting which may depend on auth context
	th.SetupRateLimitBypass()
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

	// Add auth token
	req.Header.Set("Authorization", "Bearer "+testToken)

	// Set up auth validation
	testDisplayID, _ := uuid.Parse(testDisplayIDStr)
	th.Auth.On("ValidateAccessToken", mock.Anything, testToken).Return(testDisplayID, nil)

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
	// Default rate limit configuration that allows traffic
	limit := ratelimit.Limit{
		Rate:        100,
		Period:      time.Minute,
		BurstSize:   10,
		WaitTimeout: time.Second,
	}

	// Default status showing available capacity
	status := &ratelimit.LimitStatus{
		Limit:     limit,
		Remaining: 95,
		Reset:     time.Now().Add(time.Minute),
		Period:    time.Minute,
	}

	// Set up expectations for each rate limit type
	for limitType := range validRateLimitTypes {
		// GetLimit returns the default limit
		th.RateLimit.On("GetLimit", limitType).Return(limit).Maybe()

		// Status returns success status with available capacity
		th.RateLimit.On("Status", mock.MatchedBy(func(key ratelimit.LimitKey) bool {
			return key.Type == limitType
		})).Return(status, nil).Maybe()

		// Allow always succeeds
		th.RateLimit.On("Allow", mock.Anything, mock.MatchedBy(func(key ratelimit.LimitKey) bool {
			return key.Type == limitType
		})).Return(nil).Maybe()
	}

	// Set up default expectations for any unmatched calls
	th.RateLimit.On("Status", mock.Anything).Return(status, nil).Maybe()
	th.RateLimit.On("Allow", mock.Anything, mock.Anything).Return(nil).Maybe()
}

// SetupAuthBypass configures auth service mock to bypass token validation
func (th *TestHandler) SetupAuthBypass() {
	testDisplayID, _ := uuid.Parse(testDisplayIDStr)
	// Accept any token in tests, but validate it's not empty
	th.Auth.On("ValidateAccessToken", mock.Anything, mock.MatchedBy(func(token string) bool {
		if token == "" {
			th.t.Log("Empty token received in auth validation")
			return false
		}
		return true
	})).Return(testDisplayID, nil).Maybe()
}

// ValidateJSON attempts to decode JSON into a map and verify its structure
func (th *TestHandler) ValidateJSON(body []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}
	return result, nil
}

// CleanupTest performs cleanup after each test
func (th *TestHandler) CleanupTest() {
	verifyMock := func(m *mock.Mock, name string) {
		if !m.AssertExpectations(th.t) {
			th.t.Logf("Failed expectations for %s mock:", name)
			printUnmetExpectations(th.t, m, name)
		}
	}

	// Verify all mocks with improved debugging
	verifyMock(&th.Service.Mock, "Service")
	verifyMock(&th.Activation.Mock, "Activation")
	verifyMock(&th.Auth.Mock, "Auth")
	verifyMock(&th.RateLimit.Mock, "RateLimit")
}

// printUnmetExpectations prints detailed information about unmet expectations
func printUnmetExpectations(t *testing.T, m *mock.Mock, name string) {
	// Track which expected calls were matched
	matched := make(map[string]bool)
	for _, call := range m.Calls {
		matched[mockCallString(call)] = true
	}

	// Find and report unmet expectations
	unmet := []string{}
	for _, exp := range m.ExpectedCalls {
		callStr := mockCallString(mock.Call{
			Method:    exp.Method,
			Arguments: exp.Arguments,
		})
		if !matched[callStr] {
			unmet = append(unmet, callStr)
		}
	}

	if len(unmet) > 0 {
		t.Logf("Unmet expectations for %s:", name)
		for _, call := range unmet {
			t.Logf("  - %s", call)
		}
	}

	// Log actual calls for comparison
	if len(m.Calls) > 0 {
		t.Logf("Actual calls to %s:", name)
		for _, call := range m.Calls {
			t.Logf("  - %s", mockCallString(call))
		}
	}
}

// mockCallString formats a mock call for debugging output.
// It properly handles type assertions and mock matcher arguments.
func mockCallString(call mock.Call) string {
	args := make([]string, len(call.Arguments))
	for i, arg := range call.Arguments {
		switch v := arg.(type) {
		case mock.AnythingOfTypeArgument:
			// AnythingOfTypeArgument implements fmt.Stringer
			args[i] = fmt.Sprintf("any(%s)", v)
		case fmt.Stringer:
			args[i] = fmt.Sprintf("matches(%s)", v.String())
		case nil:
			args[i] = "nil"
		default:
			args[i] = fmt.Sprintf("%v", v)
		}
	}
	return fmt.Sprintf("%s(%s)", call.Method, strings.Join(args, ", "))
}
