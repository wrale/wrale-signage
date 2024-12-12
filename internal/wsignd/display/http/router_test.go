package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

func TestRouter(t *testing.T) {
	// Create handler with mock services
	handler, mockSvc := newTestHandler()

	// Setup rate limit mocks
	mockLimitSvc := handler.rateLimit.(*mockRateLimitService)
	mockLimitSvc.On("GetLimit", mock.AnythingOfType("string")).Return(ratelimit.Limit{
		Rate:      100,
		Period:    time.Minute,
		BurstSize: 10,
	})
	mockLimitSvc.On("Allow", mock.Anything, mock.Anything).Return(nil)

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
			wantStatus:    http.StatusBadRequest, // Invalid code
			rateLimitType: "device_code",
		},
		{
			name:          "token refresh endpoint",
			method:        http.MethodPost,
			path:          "/api/v1alpha1/displays/token/refresh",
			body:          `{"refresh_token":"test"}`,
			auth:          false,
			wantStatus:    http.StatusBadRequest, // Invalid token
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
		},
		{
			name:       "readiness check endpoint",
			method:     http.MethodGet,
			path:       "/api/v1alpha1/displays/readyz",
			auth:       false,
			wantStatus: http.StatusOK,
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

			// If rate limiting expected, verify correct type
			if tt.rateLimitType != "" {
				mockLimitSvc.On("GetLimit", tt.rateLimitType).Return(ratelimit.Limit{
					Rate:      100,
					Period:    time.Minute,
					BurstSize: 10,
				}).Once()
			}

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			// Verify rate limit calls if expected
			if tt.rateLimitType != "" {
				mockLimitSvc.AssertCalled(t, "GetLimit", tt.rateLimitType)
			}
		})
	}
}

func TestRouterMiddleware(t *testing.T) {
	handler, mockSvc := newTestHandler()

	// Setup rate limit mocks
	mockLimitSvc := handler.rateLimit.(*mockRateLimitService)
	mockLimitSvc.On("GetLimit", mock.AnythingOfType("string")).Return(ratelimit.Limit{
		Rate:      100,
		Period:    time.Minute,
		BurstSize: 10,
	})
	mockLimitSvc.On("Allow", mock.Anything, mock.Anything).Return(nil)

	// Setup display service mocks
	mockSvc.On("GetByName", mock.Anything, mock.AnythingOfType("string")).Return(nil, display.ErrNotFound{ID: "unknown"})

	router := handler.Router()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "adds request id header",
			test: func(t *testing.T) {
				// Use public endpoint to avoid auth
				req := httptest.NewRequest(http.MethodPost, "/api/v1alpha1/displays/device/code", nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.NotEmpty(t, rec.Header().Get("Request-ID"))
			},
		},
		{
			name: "recovers from panic",
			test: func(t *testing.T) {
				// Create a test handler that panics
				router.HandleFunc("/api/v1alpha1/displays/panic", func(w http.ResponseWriter, r *http.Request) {
					panic("test panic")
				})

				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/panic", nil)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusInternalServerError, rec.Code)
				var resp map[string]interface{}
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, "internal error", resp["error"])
			},
		},
		{
			name: "handles context cancellation",
			test: func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				// Use public endpoint to avoid auth
				req := httptest.NewRequest(http.MethodPost, "/api/v1alpha1/displays/device/code", nil).WithContext(ctx)
				rec := httptest.NewRecorder()

				// Cancel before request
				cancel()

				router.ServeHTTP(rec, req)

				// Should return context canceled
				assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
				var resp map[string]interface{}
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, "context canceled", resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
