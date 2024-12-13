package testing

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	displayhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http"
	"github.com/wrale/wrale-signage/internal/wsignd/display/http/testing/mocks"
)

// TestHandler provides access to handler and mocks for testing
type TestHandler struct {
	Handler    *displayhttp.Handler
	Service    *mocks.Service
	Activation *mocks.ActivationService
	Auth       *mocks.AuthService
	RateLimit  *mocks.RateLimitService
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

// SetupRateLimitBypass configures rate limit mock to always allow requests
func (th *TestHandler) SetupRateLimitBypass() {
	th.RateLimit.On("GetLimit", "device_code").Return(struct{}{})
	th.RateLimit.On("GetLimit", "display").Return(struct{}{})
	th.RateLimit.On("Allow", "device_code", "test-request-id").Return(nil)
	th.RateLimit.On("Allow", "display", "test-request-id").Return(nil)
}
