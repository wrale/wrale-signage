package http_test

import (
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
	"github.com/wrale/wrale-signage/internal/wsignd/auth"
	"github.com/wrale/wrale-signage/internal/wsignd/display"
	"github.com/wrale/wrale-signage/internal/wsignd/display/activation"
	testhttp "github.com/wrale/wrale-signage/internal/wsignd/display/http/testing"
	"github.com/wrale/wrale-signage/internal/wsignd/ratelimit"
)

func TestRequestDeviceCode(t *testing.T) {
	th := testhttp.NewTestHandler()
	defer th.CleanupTest()

	// Setup standard rate limiting bypass
	th.SetupRateLimitBypass()

	// Setup mock responses
	th.Activation.On("GenerateCode", mock.Anything).Return(&activation.DeviceCode{
		DeviceCode:   "dev-code",
		UserCode:     "user-code",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		PollInterval: 5,
	}, nil)

	// Make request
	req, err := th.MockRequest(http.MethodPost, "/api/v1alpha1/displays/device/code", nil)
	assert.NoError(t, err)
	rec := httptest.NewRecorder()

	th.Handler.Router().ServeHTTP(rec, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp v1alpha1.DeviceCodeResponse
	err = json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "dev-code", resp.DeviceCode)
	assert.Equal(t, "user-code", resp.UserCode)
	assert.Equal(t, 5, resp.PollInterval)
	assert.Equal(t, "/api/v1alpha1/displays/activate", resp.VerificationURI)
}

func TestActivateDeviceCode(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*testhttp.TestHandler, uuid.UUID)
		requestBody string
		wantStatus  int
	}{
		{
			name: "successful activation",
			setupMocks: func(th *testhttp.TestHandler, testID uuid.UUID) {
				// Setup successful registration
				th.Service.On("Register",
					mock.Anything,
					"test-display",
					display.Location{
						SiteID:   "test-site",
						Zone:     "test-zone",
						Position: "main",
					},
				).Return(&display.Display{
					ID:   testID,
					Name: "test-display",
					Location: display.Location{
						SiteID:   "test-site",
						Zone:     "test-zone",
						Position: "main",
					},
					State: display.StateUnregistered,
				}, nil)

				// Setup successful activation
				th.Activation.On("ActivateCode",
					mock.Anything,
					"TEST123",
					mock.MatchedBy(func(id uuid.UUID) bool {
						return id == testID
					}),
				).Return(nil)

				// Setup token creation
				th.Auth.On("CreateToken",
					mock.Anything,
					mock.MatchedBy(func(id uuid.UUID) bool {
						return id == testID
					}),
				).Return(&auth.Token{
					AccessToken:        "access-token",
					RefreshToken:       "refresh-token",
					AccessTokenExpiry:  time.Now().Add(time.Hour),
					RefreshTokenExpiry: time.Now().Add(24 * time.Hour),
				}, nil)
			},
			requestBody: `{
				"activation_code": "TEST123",
				"name": "test-display",
				"location": {
					"site_id": "test-site",
					"zone": "test-zone",
					"position": "main"
				}
			}`,
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid activation code",
			setupMocks: func(th *testhttp.TestHandler, testID uuid.UUID) {
				// Registration succeeds
				th.Service.On("Register",
					mock.Anything,
					"test-display",
					display.Location{
						SiteID:   "test-site",
						Zone:     "test-zone",
						Position: "main",
					},
				).Return(&display.Display{
					ID:   testID,
					Name: "test-display",
					Location: display.Location{
						SiteID:   "test-site",
						Zone:     "test-zone",
						Position: "main",
					},
					State: display.StateUnregistered,
				}, nil)

				// But activation fails with not found
				th.Activation.On("ActivateCode",
					mock.Anything,
					"INVALID",
					mock.MatchedBy(func(id uuid.UUID) bool {
						return id == testID
					}),
				).Return(activation.ErrCodeNotFound)
			},
			requestBody: `{
				"activation_code": "INVALID",
				"name": "test-display",
				"location": {
					"site_id": "test-site",
					"zone": "test-zone",
					"position": "main"
				}
			}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:        "invalid request body",
			setupMocks:  func(th *testhttp.TestHandler, testID uuid.UUID) {},
			requestBody: `{invalid json`,
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := testhttp.NewTestHandler()
			testID := uuid.New()

			defer th.CleanupTest()

			// Setup rate limiting bypass
			th.SetupRateLimitBypass()

			// Setup test-specific mocks
			tt.setupMocks(th, testID)

			// Make request
			req, err := th.MockRequest(http.MethodPost, "/api/v1alpha1/displays/activate",
				strings.NewReader(tt.requestBody))
			assert.NoError(t, err)
			rec := httptest.NewRecorder()

			th.Handler.Router().ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			// Verify JSON response format if needed
			if tt.wantStatus != http.StatusOK {
				var respBody map[string]interface{}
				err = json.NewDecoder(rec.Body).Decode(&respBody)
				assert.NoError(t, err)
				assert.Contains(t, respBody, "error")
			}
		})
	}
}
