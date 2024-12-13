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
	}{
		{
			name:       "health check endpoint",
			path:       "/api/v1alpha1/displays/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "readiness check endpoint",
			path:       "/api/v1alpha1/displays/readyz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent endpoint",
			path:       "/api/v1alpha1/displays/invalid",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := testhttp.NewTestHandler()

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			th.Handler.Router().ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantStatus == http.StatusOK {
				var resp struct {
					Status string `json:"status"`
				}
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "ok", resp.Status)
			}
		})
	}
}
