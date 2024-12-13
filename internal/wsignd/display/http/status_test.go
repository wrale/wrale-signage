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
		wantBody   string
	}{
		{
			name:       "health check endpoint",
			path:       "/api/v1alpha1/displays/healthz",
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
		},
		{
			name:       "readiness check endpoint",
			path:       "/api/v1alpha1/displays/readyz",
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
		},
		{
			name:       "non-existent endpoint",
			path:       "/api/v1alpha1/displays/invalid",
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := testhttp.NewTestHandler()
			th.SetupRateLimitBypass()

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			th.Handler.Router().ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var got, want interface{}
			err := json.Unmarshal([]byte(tt.wantBody), &want)
			assert.NoError(t, err)

			err = json.Unmarshal(rec.Body.Bytes(), &got)
			assert.NoError(t, err)

			assert.Equal(t, want, got)
		})
	}
}
