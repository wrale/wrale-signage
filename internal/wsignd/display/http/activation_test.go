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
)

func TestRequestDeviceCode(t *testing.T) {
	th := testhttp.NewTestHandler(t)
	defer th.CleanupTest()

	// Setup standard rate limiting bypass
	th.SetupRateLimitBypass()

	// Setup mock responses
	th.Activation.On("GenerateCode", mock.Anything).Return(&activation.DeviceCode{
		DeviceCode:   "dev-code",
		UserCode:     "user-code",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		PollInterval: 5,
	}, nil).Once()

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
		requestBody string
		setupMocks  func(*testhttp.TestHandler, uuid.UUID)
		wantStatus  int
		wantError   *struct {
			code    string
			message string
		}
	}{
		{
			name: "successful activation",
			requestBody: `{
				"activation_code": "TEST123",
				"name": "test-display",
				"location": {
					"site_id": "test-site",
					"zone": "test-zone",
					"position": "main"
				}
			}`,
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
				}, nil).Once()

				// Setup successful activation
				th.Activation.On("ActivateCode",
					mock.Anything,
					"TEST123",
					testID,
				).Return(nil).Once()

				// Setup token creation
				th.Auth.On("CreateToken",
					mock.Anything,
					testID,
				).Return(&auth.Token{
					AccessToken:        "access-token",
					RefreshToken:       "refresh-token",
					AccessTokenExpiry:  time.Now().Add(time.Hour),
					RefreshTokenExpiry: time.Now().Add(24 * time.Hour),
				}, nil).Once()
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid activation code",
			requestBody: `{
				"activation_code": "INVALID",
				"name": "test-display",
				"location": {
					"site_id": "test-site",
					"zone": "test-zone",
					"position": "main"
				}
			}`,
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
				}, nil).Once()

				// But activation fails with not found
				th.Activation.On("ActivateCode",
					mock.Anything,
					"INVALID",
					testID,
				).Return(activation.ErrCodeNotFound).Once()
			},
			wantStatus: http.StatusNotFound,
			wantError: &struct {
				code    string
				message string
			}{
				code:    "NOT_FOUND",
				message: "activation code not found",
			},
		},
		{
			name:        "invalid request body",
			requestBody: `{invalid json`,
			setupMocks:  func(th *testhttp.TestHandler, testID uuid.UUID) {},
			wantStatus:  http.StatusBadRequest,
			wantError: &struct {
				code    string
				message string
			}{
				code:    "INVALID_INPUT",
				message: "invalid request body",
			},
		},
		{
			name: "missing required fields",
			requestBody: `{
				"activation_code": "",
				"name": "",
				"location": {}
			}`,
			setupMocks: func(th *testhttp.TestHandler, testID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
			wantError: &struct {
				code    string
				message string
			}{
				code:    "INVALID_INPUT",
				message: "activation code and display name are required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			th := testhttp.NewTestHandler(t)
			testID := uuid.New()

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

			// Assert status code first
			assert.Equal(t, tt.wantStatus, rec.Code)

			// Verify JSON response format
			var respBody map[string]interface{}
			err = json.NewDecoder(rec.Body).Decode(&respBody)
			assert.NoError(t, err)

			if tt.wantError != nil {
				assert.Equal(t, tt.wantError.code, respBody["code"], "Error code mismatch")
				assert.Equal(t, tt.wantError.message, respBody["message"], "Error message mismatch")
			} else {
				assert.Contains(t, respBody, "display", "Response should contain display info")
				assert.Contains(t, respBody, "auth", "Response should contain auth info")

				auth, ok := respBody["auth"].(map[string]interface{})
				assert.True(t, ok, "Auth should be a map")
				assert.Contains(t, auth, "access_token")
				assert.Contains(t, auth, "refresh_token")
			}

			// Verify all mock expectations were met
			th.CleanupTest()
		})
	}
}
