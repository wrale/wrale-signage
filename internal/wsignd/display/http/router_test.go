package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

func TestRouter(t *testing.T) {
	// Create handler with mock services
	handler, mockSvc := newTestHandler()

	// Setup rate limit mocks with specific expectations
	mockLimitSvc := handler.rateLimit.(*mockRateLimitService)

	// Map of endpoint types to rate limits
	rateLimits := map[string]ratelimit.Limit{
		"api_request": {
			Rate:      100,
			Period:    time.Minute,
			BurstSize: 10,
		},
		"device_code": {
			Rate:      100,
			Period:    time.Minute,
			BurstSize: 10,
		},
		"token_refresh": {
			Rate:      100,
			Period:    time.Minute,
			BurstSize: 10,
		},
		"websocket": {
			Rate:      100,
			Period:    time.Minute,
			BurstSize: 10,
		},
	}

	for limitType, limit := range rateLimits {
		mockLimitSvc.On("GetLimit", limitType).Return(limit).Maybe()
	}
	mockLimitSvc.On("Allow", mock.Anything, mock.Anything).Return(nil)

	// Setup activation service mocks
	mockActSvc := handler.activation.(*mockActivationService)
	mockActSvc.On("GenerateCode", mock.Anything).Return(&activation.DeviceCode{
		DeviceCode:   "dev-code",
		UserCode:     "user-code",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		PollInterval: 5,
	}, nil)

	mockActSvc.On("ActivateCode", mock.Anything, mock.AnythingOfType("string")).Return(uuid.Nil, activation.ErrCodeNotFound)

	// Setup display service mocks
	mockSvc.On("GetByName", mock.Anything, mock.AnythingOfType("string")).Return(nil, display.ErrNotFound{ID: "unknown"})

	router := handler.Router()

	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		auth          bool // Whether to include auth header
		wantStatus    int
		rateLimitType string // Rate limit type to expect
	}{
		// Public endpoints (no auth required)
		{
			name:          "display registration endpoint",
			method:        http.MethodPost,
			path:          "/api/v1alpha1/displays",
			body:          `{"name":"test","site_id":"hq","zone":"lobby"}`,
			auth:          false,
			wantStatus:    http.StatusBadRequest, // Invalid JSON
			rateLimitType: "api_request",
		},
		{
			name:          "device code request endpoint",
			method:        http.MethodPost,
			path:          "/api/v1alpha1/displays/device/code",
			auth:          false,
			wantStatus:    http.StatusOK,
			rateLimitType: "device_code",
		},
		{
			name:          "device activation endpoint",
			method:        http.MethodPost,
			path:          "/api/v1alpha1/displays/activate",
			body:          `{"code":"TEST123"}`,
			auth:          false,
			wantStatus:    http.StatusNotFound, // Invalid code returns 404
			rateLimitType: "device_code",
		},
		{
			name:          "token refresh endpoint",
			method:        http.MethodPost,
			path:          "/api/v1alpha1/displays/token/refresh",
			body:          `{"refresh_token":"test"}`,
			auth:          false,
			wantStatus:    http.StatusUnauthorized, // Missing Authorization header
			rateLimitType: "token_refresh",
		},

		// Protected endpoints (auth required)
		{
			name:          "display get endpoint - no auth",
			method:        http.MethodGet,
			path:          "/api/v1alpha1/displays/123",
			auth:          false,
			wantStatus:    http.StatusUnauthorized,
			rateLimitType: "api_request",
		},
		{
			name:          "display activate endpoint - no auth",
			method:        http.MethodPut,
			path:          "/api/v1alpha1/displays/123/activate",
			auth:          false,
			wantStatus:    http.StatusUnauthorized,
			rateLimitType: "api_request",
		},
		{
			name:          "display last seen endpoint - no auth",
			method:        http.MethodPut,
			path:          "/api/v1alpha1/displays/123/last-seen",
			auth:          false,
			wantStatus:    http.StatusUnauthorized,
			rateLimitType: "api_request",
		},
		{
			name:          "websocket endpoint - no auth",
			method:        http.MethodGet,
			path:          "/api/v1alpha1/displays/ws",
			auth:          false,
			wantStatus:    http.StatusUnauthorized,
			rateLimitType: "websocket",
		},

		// Health check endpoints (bypass rate limits)
		{
			name:       "health check endpoint",
			method:     http.MethodGet,
			path:       "/api/v1alpha1/displays/healthz",
			auth:       false,
			wantStatus: http.StatusOK,
			// No rate limit type - health checks bypass limits
		},
		{
			name:       "readiness check endpoint",
			method:     http.MethodGet,
			path:       "/api/v1alpha1/displays/readyz",
			auth:       false,
			wantStatus: http.StatusOK,
			// No rate limit type - health checks bypass limits
		},

		// Invalid routes
		{
			name:       "non-existent endpoint returns 404",
			method:     http.MethodGet,
			path:       "/api/v1alpha1/non-existent",
			auth:       false,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			if tt.auth {
				req.Header.Set("Authorization", "Bearer test-token")
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			// Verify rate limit calls if expected
			if tt.rateLimitType != "" {
				mockLimitSvc.AssertExpectations(t)
			}
		})
	}
}

func TestRouterMiddleware(t *testing.T) {
	handler, _ := newTestHandler()
	mockActSvc := handler.activation.(*mockActivationService)

	// Setup rate limit mocks
	mockLimitSvc := handler.rateLimit.(*mockRateLimitService)
	mockLimitSvc.On("GetLimit", "device_code").Return(ratelimit.Limit{
		Rate:      100,
		Period:    time.Minute,
		BurstSize: 10,
	}).Maybe()
	mockLimitSvc.On("Allow", mock.Anything, mock.Anything).Return(nil)

	// Setup activation service mocks for device code test
	mockActSvc.On("GenerateCode", mock.Anything).Return(&activation.DeviceCode{
		DeviceCode:   "dev-code",
		UserCode:     "user-code",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		PollInterval: 5,
	}, nil)

	router := handler.Router()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "adds request id header",
			test: func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/api/v1alpha1/displays/device/code", nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.NotEmpty(t, rec.Header().Get("Request-ID"))
				assert.Equal(t, http.StatusOK, rec.Code)

				var resp v1alpha1.DeviceCodeResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.NotEmpty(t, resp.DeviceCode)
			},
		},
		{
			name: "recovers from panic",
			test: func(t *testing.T) {
				// Add panic handler under test
				router.HandleFunc("/api/v1alpha1/displays/panic", func(w http.ResponseWriter, r *http.Request) {
					panic("test panic")
				})

				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/panic", nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusInternalServerError, rec.Code)

				var resp struct {
					Error string `json:"error"`
				}
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "internal error", resp.Error)
			},
		},
		{
			name: "handles context cancellation",
			test: func(t *testing.T) {
				// Create a slow handler that will be canceled
				router.HandleFunc("/api/v1alpha1/displays/slow", func(w http.ResponseWriter, r *http.Request) {
					select {
					case <-r.Context().Done():
						// Context canceled, write error response
						w.WriteHeader(http.StatusServiceUnavailable)
						json.NewEncoder(w).Encode(map[string]string{
							"error": "context canceled",
						})
					case <-time.After(time.Second):
						// Should not reach here
						t.Error("handler not canceled")
					}
				})

				// Create canceled context
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately

				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/slow", nil)
				req = req.WithContext(ctx)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

				var resp struct {
					Error string `json:"error"`
				}
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "context canceled", resp.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
