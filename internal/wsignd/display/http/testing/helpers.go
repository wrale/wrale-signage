package testing

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
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
	mockSvc := &mocks.Service{}
	mockActSvc := &mocks.ActivationService{}
	mockAuthSvc := &mocks.AuthService{}
	mockRateLimitSvc := &mocks.RateLimitService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	handler := displayhttp.NewHandler(mockSvc, mockActSvc, mockAuthSvc, mockRateLimitSvc, logger)

	return &TestHandler{
		Handler:    handler,
		Service:    mockSvc,
		Activation: mockActSvc,
		Auth:       mockAuthSvc,
		RateLimit:  mockRateLimitSvc,
	}
}

// MockChiContext adds a chi routing context to the request
func MockChiContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chiCtx := chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
		next.ServeHTTP(w, r)
	})
}

// WriteJSON is a test helper that matches the production writeJSON behavior
func WriteJSON(w http.ResponseWriter, status int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}
