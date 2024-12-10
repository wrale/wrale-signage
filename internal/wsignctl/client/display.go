package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// DisplayActivation represents the status of a display activation process
type DisplayActivation struct {
	// DeviceCode is the opaque token used for verification
	DeviceCode string `json:"device_code"`
	// UserCode is the human-readable code shown on the display
	UserCode string `json:"user_code"`
	// VerificationURI is where administrators should go to activate the display
	VerificationURI string `json:"verification_uri"`
	// VerificationURIComplete includes the user code for direct access
	VerificationURIComplete string `json:"verification_uri_complete"`
	// ExpiresIn is how many seconds until the codes expire
	ExpiresIn int `json:"expires_in"`
	// Interval is how many seconds to wait between polling
	Interval int `json:"interval"`
}

// ActivateDisplay initiates the display activation process and waits for completion.
// It implements the OAuth 2.0 Device Authorization Flow as specified in RFC 8628.
func (c *Client) ActivateDisplay(ctx context.Context, display *v1alpha1.Display) error {
	// Request activation codes
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1alpha1/displays/activate", nil)
	if err != nil {
		return fmt.Errorf("error requesting activation: %w", err)
	}
	defer resp.Body.Close()

	var activation DisplayActivation
	if err := json.NewDecoder(resp.Body).Decode(&activation); err != nil {
		return fmt.Errorf("error decoding activation response: %w", err)
	}

	// Display the activation code to the user
	fmt.Printf("\nDisplay shows code: %s\n", activation.UserCode)
	fmt.Printf("Visit %s to activate\n\n", activation.VerificationURI)

	// Poll for completion
	ticker := time.NewTicker(time.Duration(activation.Interval) * time.Second)
	defer ticker.Stop()

	expiration := time.Now().Add(time.Duration(activation.ExpiresIn) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(expiration) {
				return fmt.Errorf("activation code expired")
			}

			// Check activation status
			resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1alpha1/displays/activate/poll", map[string]string{
				"device_code": activation.DeviceCode,
			})
			if err != nil {
				// Check if this is just a pending status
				if resp != nil && resp.StatusCode == http.StatusAccepted {
					continue
				}
				return fmt.Errorf("error checking activation status: %w", err)
			}
			defer resp.Body.Close()

			// Activation successful - get the activated display details
			var activated v1alpha1.Display
			if err := json.NewDecoder(resp.Body).Decode(&activated); err != nil {
				return fmt.Errorf("error decoding activated display: %w", err)
			}

			// Update the provided display struct with activated details
			*display = activated
			return nil
		}
	}
}
