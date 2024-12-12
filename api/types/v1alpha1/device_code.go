package v1alpha1

// DeviceCodeResponse represents the server's response to a device code request
type DeviceCodeResponse struct {
	// DeviceCode is the opaque code for verification
	DeviceCode string `json:"device_code"`
	// UserCode is the code shown to user (e.g., "WDJC-XYZK")
	UserCode string `json:"user_code"`
	// ExpiresIn is seconds until the codes expire
	ExpiresIn int `json:"expires_in"`
	// PollInterval is how often the device should poll for activation
	PollInterval int `json:"poll_interval"`
	// VerificationURI is where users go to activate the device
	VerificationURI string `json:"verification_uri"`
}

// DisplayRegistrationResponse is returned when a display is successfully registered
type DisplayRegistrationResponse struct {
	// Display holds the registered display information
	Display *Display `json:"display"`
	// Auth holds authentication tokens for the display
	Auth *DisplayAuthTokens `json:"auth"`
}

// DisplayAuthTokens contains authentication tokens for a display
type DisplayAuthTokens struct {
	// AccessToken is used for API and WebSocket authentication
	AccessToken string `json:"access_token"`
	// RefreshToken is used to obtain new access tokens
	RefreshToken string `json:"refresh_token"`
	// TokenType is always "Bearer"
	TokenType string `json:"token_type"`
	// ExpiresIn is seconds until the access token expires
	ExpiresIn int `json:"expires_in"`
	// RefreshExpiresIn is seconds until the refresh token expires
	RefreshExpiresIn int `json:"refresh_expires_in"`
}
