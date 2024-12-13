package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	testhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http/testing"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

func TestRouteAuthentication(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		requiresAuth bool
		wantStatus   int
	}{
		// Public endpoints
		{
			name:         "healthz endpoint",
			method:       http.MethodGet,
			path:         "/api/v1alpha1/displays/healthz",
			requiresAuth: false,
			wantStatus:   http.StatusOK,
		},
		{
			name:         "readyz endpoint",
			method:       http.MethodGet,
			path:         "/api/v1alpha1/displays/readyz",
			requiresAuth: false,
			wantStatus:   http.StatusOK,
		},
		{
			name:         "device code request",
			method:       http.MethodPost,
			path:         "/api/v1alpha1/displays/device/code",
			requiresAuth: false,
			wantStatus:   http.StatusOK,
		},
		{
			name:         "device activation",
			method:       http.MethodPost,
			path:         "/api/v1alpha1/displays/activate",
			requiresAuth: false,
			wantStatus:   http.StatusBadRequest, // Invalid JSON body
		},
		// Protected endpoints
		{
			name:         "get display",
			method:       http.MethodGet,
			path:         "/api/v1alpha1/displays/123",
			requiresAuth: true,
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "activate display",
			method:       http.MethodPut,
			path:         "/api/v1alpha1/displays/123/activate",
			requiresAuth: true,
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "update last seen",
			method:       http.MethodPut,
			path:         "/api/v1alpha1/displays/123/last-seen",
			requiresAuth: true,
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "websocket connection",
			method:       http.MethodGet,
			path:         "/api/v1alpha1/displays/ws",
			requiresAuth: true,
			wantStatus:   http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := testhttp.NewTestHandler()

			// Setup rate limiting for all routes
			th.RateLimit.On("GetLimit", mock.Anything).Return(ratelimit.Limit{
				Rate:      100,
				Period:    time.Minute,
				BurstSize: 10,
			}).Maybe()
			th.RateLimit.On("Allow", mock.Anything, mock.Anything).Return(nil).Maybe()

			// Setup activation service mock for device code tests
			if tt.path == "/api/v1alpha1/displays/device/code" {
				th.Activation.On("GenerateCode", mock.Anything).Return(testDeviceCode(), nil)
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			th.Handler.Router().ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRoutingMiddleware(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "adds request id header",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler()
				th.Activation.On("GenerateCode", mock.Anything).Return(testDeviceCode(), nil)
				th.RateLimit.On("GetLimit", mock.Anything).Return(ratelimit.Limit{})
				th.RateLimit.On("Allow", mock.Anything, mock.Anything).Return(nil)

				req := httptest.NewRequest(http.MethodPost, "/api/v1alpha1/displays/device/code", nil)
				rec := httptest.NewRecorder()

				th.Handler.Router().ServeHTTP(rec, req)

				assert.NotEmpty(t, rec.Header().Get("Request-ID"))
				assert.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "handles context cancellation",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler()
				router := th.Handler.Router()

				// Create a slow handler that will be canceled
				router.HandleFunc("/api/v1alpha1/displays/slow", func(w http.ResponseWriter, r *http.Request) {
					select {
					case <-r.Context().Done():
						w.WriteHeader(http.StatusServiceUnavailable)
					case <-time.After(time.Second):
						t.Error("handler not canceled")
					}
				})

				req := httptest.NewRequest(http.MethodGet, "/api/v1alpha1/displays/slow", nil)
				req = req.WithContext(canceledContext())
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)
				assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// Helper function to create test device code
func testDeviceCode() *activation.DeviceCode {
	return &activation.DeviceCode{
		DeviceCode:   "dev-code",
		UserCode:     "user-code",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		PollInterval: 5,
	}
}

// Helper function to create canceled context
func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
