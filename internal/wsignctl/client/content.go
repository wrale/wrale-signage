package client

import (
	"context"
	"fmt"
	"time"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

type Content struct {
	Name             string            `json:"name"`
	URL              string            `json:"url"`
	Type             string            `json:"type"`
	PlaybackDuration time.Duration     `json:"playbackDuration"`
	Properties       map[string]string `json:"properties,omitempty"`
}

// CreateContent creates new content with the given configuration
func (c *Client) CreateContent(ctx context.Context, name, url string, duration time.Duration, contentType string, properties map[string]string) error {
	request := &v1alpha1.ContentSource{
		TypeMeta: v1alpha1.TypeMeta{
			APIVersion: "wrale.io/v1alpha1",
			Kind:       "ContentSource",
		},
		ObjectMeta: v1alpha1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.ContentSourceSpec{
			URL:              url,
			Type:             contentType,
			PlaybackDuration: duration,
			Properties:       properties,
		},
	}

	if err := request.Spec.Validate(); err != nil {
		return fmt.Errorf("invalid content configuration: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1alpha1/content", request)
	if err != nil {
		return fmt.Errorf("failed to create content: %w", err)
	}
	return closeBody(resp.Body, nil)
}

// UpdateContent updates existing content configuration
func (c *Client) UpdateContent(ctx context.Context, name, url string, duration time.Duration, contentType string, properties map[string]string, remove []string) error {
	update := v1alpha1.ContentSourceUpdate{
		Properties: properties,
	}

	if url != "" {
		update.URL = &url
	}
	if duration != 0 {
		update.PlaybackDuration = &duration
	}

	resp, err := c.doRequest(ctx, "PUT", "/api/v1alpha1/content/"+name, update)
	if err != nil {
		return fmt.Errorf("failed to update content: %w", err)
	}
	return closeBody(resp.Body, nil)
}

// RemoveContent removes content by name
func (c *Client) RemoveContent(ctx context.Context, name string) error {
	resp, err := c.doRequest(ctx, "DELETE", "/api/v1alpha1/content/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to remove content: %w", err)
	}
	return closeBody(resp.Body, nil)
}

// ListContent retrieves all content sources
func (c *Client) ListContent(ctx context.Context) ([]v1alpha1.ContentSource, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1alpha1/content", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list content: %w", err)
	}
	defer resp.Body.Close()

	var list v1alpha1.ContentSourceList
	if err := decodeResponse(resp, &list); err != nil {
		return nil, err
	}

	return list.Items, nil
}
