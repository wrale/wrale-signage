package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	testhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http/testing"
)

func TestStatusEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   map[string]interface{}
	}{
		{
			name:       "health check endpoint",
			path:       "/api/v1alpha1/displays/healthz",
			wantStatus: http.StatusOK,
			wantBody: map[string]interface{}{
				"status": "ok",
			},
		},
		{
			name:       "readiness check endpoint",
			path:       "/api/v1alpha1/displays/readyz",
			wantStatus: http.StatusOK,
			wantBody: map[string]interface{}{
				"status": "ok",
			},
		},
		{
			name:       "non-existent endpoint",
			path:       "/api/v1alpha1/displays/invalid",
			wantStatus: http.StatusNotFound,
			wantBody: map[string]interface{}{
				"error": "not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := testhttp.NewTestHandler(t)
			defer th.CleanupTest()

			// Setup standard rate limiting bypass
			th.SetupRateLimitBypass()

			// Make request
			req, err := th.MockRequest(http.MethodGet, tt.path, nil)
			assert.NoError(t, err)
			rec := httptest.NewRecorder()

			th.Handler.Router().ServeHTTP(rec, req)

			// Verify status code
			assert.Equal(t, tt.wantStatus, rec.Code)

			// Verify JSON response
			var got map[string]interface{}
			err = json.NewDecoder(rec.Body).Decode(&got)
			assert.NoError(t, err, "failed to decode response body")
			assert.Equal(t, tt.wantBody, got, "response body mismatch")
		})
	}
}
