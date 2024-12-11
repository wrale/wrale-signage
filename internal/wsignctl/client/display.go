package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// closeBody safely closes a response body and wraps any error with the original error
func closeBody(body io.ReadCloser, err error) error {
	if cerr := body.Close(); cerr != nil {
		if err != nil {
			return fmt.Errorf("original error: %v, close error: %v", err, cerr)
		}
		return cerr
	}
	return err
}

// GetDisplay retrieves a display by ID
func (c *Client) GetDisplay(ctx context.Context, id string) (*v1alpha1.Display, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1alpha1/displays/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get display: %w", err)
	}
	defer resp.Body.Close()

	var display v1alpha1.Display
	if err := decodeResponse(resp, &display); err != nil {
		return nil, closeBody(resp.Body, err)
	}

	return &display, closeBody(resp.Body, nil)
}

// ListDisplays retrieves displays matching the given selector
func (c *Client) ListDisplays(ctx context.Context, selector v1alpha1.DisplaySelector) ([]v1alpha1.Display, error) {
	// Build query parameters
	u := url.Values{}
	if selector.SiteID != "" {
		u.Set("siteId", selector.SiteID)
	}
	if selector.Zone != "" {
		u.Set("zone", selector.Zone)
	}
	if selector.Position != "" {
		u.Set("position", selector.Position)
	}

	path := "/api/v1alpha1/displays"
	if len(u) > 0 {
		path += "?" + u.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list displays: %w", err)
	}
	defer resp.Body.Close()

	var displays []v1alpha1.Display
	if err := decodeResponse(resp, &displays); err != nil {
		return nil, closeBody(resp.Body, err)
	}

	return displays, closeBody(resp.Body, nil)
}

// CreateDisplay creates a new display
func (c *Client) CreateDisplay(ctx context.Context, name string, display *v1alpha1.Display) error {
	display.Name = name
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1alpha1/displays", display)
	if err != nil {
		return fmt.Errorf("failed to create display: %w", err)
	}
	return closeBody(resp.Body, nil)
}

// UpdateDisplay updates an existing display's location and properties
func (c *Client) UpdateDisplay(ctx context.Context, name string, location *v1alpha1.DisplayLocation, addProps map[string]string, removeProps []string) error {
	update := &v1alpha1.DisplayUpdateRequest{
		Location:   location,
		Properties: addProps,
	}

	resp, err := c.doRequest(ctx, http.MethodPut, "/api/v1alpha1/displays/"+name, update)
	if err != nil {
		return fmt.Errorf("failed to update display: %w", err)
	}
	return closeBody(resp.Body, nil)
}

// DeleteDisplay deletes a display
func (c *Client) DeleteDisplay(ctx context.Context, name string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "/api/v1alpha1/displays/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to delete display: %w", err)
	}
	return closeBody(resp.Body, nil)
}

// ActivateDisplay activates a display using its registration information
func (c *Client) ActivateDisplay(ctx context.Context, req *v1alpha1.DisplayRegistrationRequest) (*v1alpha1.Display, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1alpha1/displays", &v1alpha1.DisplayRegistrationRequest{
		Name:           req.Name,
		Location:       req.Location,
		ActivationCode: req.ActivationCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register display: %w", err)
	}
	defer resp.Body.Close()

	var regResp v1alpha1.DisplayRegistrationResponse
	if err := decodeResponse(resp, &regResp); err != nil {
		return nil, closeBody(resp.Body, fmt.Errorf("failed to decode registration response: %w", err))
	}

	// Then activate using the ID
	activateResp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1alpha1/displays/%s/activate", regResp.Display.ObjectMeta.ID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to activate display: %w", err)
	}
	defer activateResp.Body.Close()

	// Parse activation response
	var activated v1alpha1.Display
	if err := decodeResponse(activateResp, &activated); err != nil {
		return nil, closeBody(activateResp.Body, fmt.Errorf("failed to decode activation response: %w", err))
	}

	return &activated, nil
}