package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http/testing"
)

func TestStatusEndpoints(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		needsAuth      bool // Health/readiness don't need auth
		wantStatus     int
		wantBody       map[string]interface{}
		setupAuth      bool
		setupRateLimit bool
	}{
		{
			name:       "health check endpoint",
			path:       "/api/v1alpha1/displays/healthz",
			needsAuth:  false,
			wantStatus: http.StatusOK,
			wantBody: map[string]interface{}{
				"status": "ok",
			},
			setupRateLimit: true,
		},
		{
			name:       "readiness check endpoint",
			path:       "/api/v1alpha1/displays/readyz",
			needsAuth:  false,
			wantStatus: http.StatusOK,
			wantBody: map[string]interface{}{
				"status": "ok",
			},
			setupRateLimit: true,
		},
		{
			name:       "non-existent endpoint",
			path:       "/api/v1alpha1/displays/invalid",
			needsAuth:  true, // Protected routes require auth
			wantStatus: http.StatusNotFound,
			wantBody: map[string]interface{}{
				"error": "not found",
			},
			setupAuth:      true,
			setupRateLimit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler with mocks
			th := testhttp.NewTestHandler(t)
			defer th.CleanupTest()

			// Set up rate limiting if needed
			if tt.setupRateLimit {
				th.SetupRateLimitBypass()
			}

			// Set up auth bypass if needed
			if tt.setupAuth {
				th.SetupAuthBypass()
			}

			// Create request
			var req *http.Request
			var err error
			if tt.needsAuth {
				req, err = th.MockAuthorizedRequest(http.MethodGet, tt.path, nil)
			} else {
				req, err = th.MockRequest(http.MethodGet, tt.path, nil)
			}
			require.NoError(t, err, "failed to create request")

			// Make request
			rec := httptest.NewRecorder()
			th.Handler.Router().ServeHTTP(rec, req)

			// Verify status code
			assert.Equal(t, tt.wantStatus, rec.Code)

			// Verify JSON response
			var got map[string]interface{}
			err = json.NewDecoder(rec.Body).Decode(&got)
			require.NoError(t, err, "failed to decode response body")
			assert.Equal(t, tt.wantBody, got, "response body mismatch")

			// Verify headers
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			assert.NotEmpty(t, rec.Header().Get("Request-Id"))
		})
	}
}
