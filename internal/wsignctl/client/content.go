package client

import (
	"context"
	"fmt"
	"time"
)

type Content struct {
	Name       string            `json:"name"`
	Path       string            `json:"path"`
	Type       string            `json:"type"`
	Duration   string            `json:"duration"`
	Properties map[string]string `json:"properties,omitempty"`
}

// CreateContent creates new content with the given configuration
func (c *Client) CreateContent(ctx context.Context, name, path string, duration time.Duration, contentType string, properties map[string]string) error {
	request := Content{
		Name:       name,
		Path:       path,
		Duration:   duration.String(),
		Type:       contentType,
		Properties: properties,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1alpha1/content", request)
	if err != nil {
		return fmt.Errorf("failed to create content: %w", err)
	}
	return closeBody(resp.Body, nil)
}

// UpdateContent updates existing content configuration
func (c *Client) UpdateContent(ctx context.Context, name, path string, duration time.Duration, contentType string, properties map[string]string, remove []string) error {
	request := struct {
		Path        string            `json:"path,omitempty"`
		Duration    string            `json:"duration,omitempty"`
		Type        string            `json:"type,omitempty"`
		Properties  map[string]string `json:"properties,omitempty"`
		RemoveProps []string          `json:"removeProperties,omitempty"`
	}{
		Path:        path,
		Duration:    duration.String(),
		Type:        contentType,
		Properties:  properties,
		RemoveProps: remove,
	}

	resp, err := c.doRequest(ctx, "PUT", "/api/v1alpha1/content/"+name, request)
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
func (c *Client) ListContent(ctx context.Context) ([]Content, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1alpha1/content", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list content: %w", err)
	}
	defer resp.Body.Close()

	var content []Content
	if err := decodeResponse(resp, &content); err != nil {
		return nil, err
	}

	return content, nil
}
