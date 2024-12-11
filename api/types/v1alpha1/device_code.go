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
