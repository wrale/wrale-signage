// Package client provides an HTTP client for interacting with the Wrale Signage API
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// Client provides methods for interacting with the Wrale Signage API
type Client struct {
	// baseURL is the root URL for all API requests
	baseURL string
	// httpClient is the underlying HTTP client
	httpClient *http.Client
	// token is the authentication token
	token string
}

// ClientOption configures a Client
type ClientOption func(*Client)

// WithToken sets the authentication token
func WithToken(token string) ClientOption {
	return func(c *Client) {
		c.token = token
	}
}

// WithTLSConfig sets custom TLS configuration
func WithTLSConfig(config *tls.Config) ClientOption {
	return func(c *Client) {
		tr := &http.Transport{
			TLSClientConfig: config,
		}
		c.httpClient = &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		}
	}
}

// NewClient creates a new API client
func NewClient(baseURL string, options ...ClientOption) (*Client, error) {
	// Validate and normalize base URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if u.Path != "" {
		u.Path = ""
	}

	// Create client with defaults
	c := &Client{
		baseURL: u.String(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Apply options
	for _, opt := range options {
		opt(c)
	}

	return c, nil
}

// doRequest performs an HTTP request with automatic error handling
func (c *Client) doRequest(ctx context.Context, method, pathStr string, body interface{}) (*http.Response, error) {
	// Build full URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path.Join(u.Path, pathStr)

	// Create request body if needed
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error encoding request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Perform request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}

	// Check for API errors
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		var apiErr struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("HTTP %d: unable to decode error response", resp.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiErr.Message)
	}

	return resp, nil
}

// GetDisplay retrieves a display by ID
func (c *Client) GetDisplay(ctx context.Context, id string) (*v1alpha1.Display, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1alpha1/displays/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var display v1alpha1.Display
	if err := json.NewDecoder(resp.Body).Decode(&display); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &display, nil
}

// ListDisplays retrieves all displays
func (c *Client) ListDisplays(ctx context.Context) ([]v1alpha1.Display, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1alpha1/displays", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var displays []v1alpha1.Display
	if err := json.NewDecoder(resp.Body).Decode(&displays); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return displays, nil
}

// CreateDisplay creates a new display
func (c *Client) CreateDisplay(ctx context.Context, display *v1alpha1.Display) error {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1alpha1/displays", display)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// UpdateDisplay updates an existing display
func (c *Client) UpdateDisplay(ctx context.Context, display *v1alpha1.Display) error {
	resp, err := c.doRequest(ctx, http.MethodPut, "/api/v1alpha1/displays/"+display.Name, display)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// DeleteDisplay deletes a display
func (c *Client) DeleteDisplay(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "/api/v1alpha1/displays/"+id, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
