package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	testhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http/testing"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "recovers from panic",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler(t)
				defer th.CleanupTest()

				th.SetupRateLimitBypass()

				// Add panic handler
				router := th.Handler.Router()
				router.HandleFunc("/test/panic", func(w http.ResponseWriter, r *http.Request) {
					panic("test panic")
				})

				req, err := th.MockRequest(http.MethodGet, "/test/panic", nil)
				assert.NoError(t, err)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusInternalServerError, rec.Code)

				var resp struct {
					Error string `json:"error"`
				}
				err = json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "internal error", resp.Error)
			},
		},
		{
			name: "adds request id to context",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler(t)
				defer th.CleanupTest()

				th.SetupRateLimitBypass()

				// Add test handler
				router := th.Handler.Router()
				router.Get("/test/id", func(w http.ResponseWriter, r *http.Request) {
					// Request ID should be propagated and added to response headers
					w.WriteHeader(http.StatusOK)
				})

				req, err := th.MockRequest(http.MethodGet, "/test/id", nil)
				assert.NoError(t, err)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)

				assert.Equal(t, http.StatusOK, rec.Code)
				assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
				assert.Equal(t, rec.Header().Get("Request-ID"), rec.Header().Get("X-Request-ID"))
			},
		},
		{
			name: "enforces rate limits",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler(t)
				defer th.CleanupTest()

				// Configure rate limit mock to deny request
				th.RateLimit.On("GetLimit", "api_request").Return(ratelimit.Limit{
					Rate:      1,
					Period:    time.Minute,
					BurstSize: 1,
				})

				// Expect rate limit check with proper key
				th.RateLimit.On("Allow", mock.Anything, mock.MatchedBy(func(key ratelimit.LimitKey) bool {
					return key.Type == "api_request" && key.RemoteIP != ""
				})).Return(ratelimit.ErrLimitExceeded)

				req, err := th.MockRequest(http.MethodGet, "/api/v1alpha1/displays/123", nil)
				assert.NoError(t, err)
				rec := httptest.NewRecorder()

				th.Handler.Router().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusTooManyRequests, rec.Code)

				// Verify JSON response
				var respBody map[string]interface{}
				err = json.NewDecoder(rec.Body).Decode(&respBody)
				assert.NoError(t, err)
				assert.Contains(t, respBody, "error")
				assert.Contains(t, respBody["error"], "rate limit")
			},
		},
		{
			name: "handles context cancellation",
			test: func(t *testing.T) {
				th := testhttp.NewTestHandler(t)
				defer th.CleanupTest()

				th.SetupRateLimitBypass()

				// Add handler that blocks until context is cancelled
				router := th.Handler.Router()
				router.HandleFunc("/test/slow", func(w http.ResponseWriter, r *http.Request) {
					<-r.Context().Done()
					w.WriteHeader(http.StatusServiceUnavailable)
				})

				// Create cancellable request
				ctx, cancel := context.WithCancel(context.Background())
				req, err := th.MockRequest(http.MethodGet, "/test/slow", nil)
				assert.NoError(t, err)
				req = req.WithContext(ctx)
				rec := httptest.NewRecorder()

				// Cancel context after a short delay
				go func() {
					time.Sleep(100 * time.Millisecond)
					cancel()
				}()

				router.ServeHTTP(rec, req)
				assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
