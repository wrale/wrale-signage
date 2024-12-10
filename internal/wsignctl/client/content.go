package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// AddContentSource creates a new content source in the system. It takes a complete ContentSource
// object that specifies all required fields including name, URL, and content type.
// The server will validate the input and ensure the name is unique before creating
// the content source.
func (c *Client) AddContentSource(ctx context.Context, source *v1alpha1.ContentSource) error {
	resp, err := c.doRequest(ctx, "POST", "/api/v1alpha1/content", source)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// UpdateContentSource updates an existing content source identified by name. The update
// parameter specifies which fields to modify - only non-nil fields will be updated.
// This allows for partial updates without affecting other fields.
func (c *Client) UpdateContentSource(ctx context.Context, name string, update *v1alpha1.ContentSourceUpdate) error {
	resp, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/api/v1alpha1/content/%s", name), update)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// RemoveContentSource deletes a content source from the system. If force is false,
// the operation will fail if any redirect rules reference this content source.
// Setting force to true will delete the content source and invalidate any
// referring rules.
func (c *Client) RemoveContentSource(ctx context.Context, name string, force bool) error {
	path := fmt.Sprintf("/api/v1alpha1/content/%s", name)
	if force {
		path += "?force=true"
	}
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ListContentSources retrieves all content sources in the system. The results can be
// filtered server-side by passing query parameters, though this basic implementation
// returns all sources.
func (c *Client) ListContentSources(ctx context.Context) ([]v1alpha1.ContentSource, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1alpha1/content", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list v1alpha1.ContentSourceList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return list.Items, nil
}

// GetContentSource retrieves a single content source by name. Returns an error
// if the content source doesn't exist.
func (c *Client) GetContentSource(ctx context.Context, name string) (*v1alpha1.ContentSource, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1alpha1/content/%s", name), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var source v1alpha1.ContentSource
	if err := json.NewDecoder(resp.Body).Decode(&source); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &source, nil
}
