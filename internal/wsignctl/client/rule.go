package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// AddRedirectRule creates a new redirect rule
func (c *Client) AddRedirectRule(ctx context.Context, rule *v1alpha1.RedirectRule) error {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1alpha1/rules", rule)
	if err != nil {
		return fmt.Errorf("failed to create redirect rule: %w", err)
	}
	resp.Body.Close()
	return nil
}

// ListRedirectRules retrieves all redirect rules
func (c *Client) ListRedirectRules(ctx context.Context) ([]v1alpha1.RedirectRule, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1alpha1/rules", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list redirect rules: %w", err)
	}
	defer resp.Body.Close()

	var rules []v1alpha1.RedirectRule
	if err := decodeResponse(resp, &rules); err != nil {
		return nil, err
	}

	return rules, nil
}

// UpdateRedirectRule updates an existing redirect rule
func (c *Client) UpdateRedirectRule(ctx context.Context, rule *v1alpha1.RedirectRule) error {
	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1alpha1/rules/%s", rule.Name), rule)
	if err != nil {
		return fmt.Errorf("failed to update redirect rule: %w", err)
	}
	resp.Body.Close()
	return nil
}

// DeleteRedirectRule deletes a redirect rule
func (c *Client) DeleteRedirectRule(ctx context.Context, name string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1alpha1/rules/%s", name), nil)
	if err != nil {
		return fmt.Errorf("failed to delete redirect rule: %w", err)
	}
	resp.Body.Close()
	return nil
}

// ReorderRedirectRules updates the order of redirect rules
func (c *Client) ReorderRedirectRules(ctx context.Context, names []string) error {
	type reorderRequest struct {
		Names []string `json:"names"`
	}

	req := reorderRequest{Names: names}
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1alpha1/rules/reorder", req)
	if err != nil {
		return fmt.Errorf("failed to reorder redirect rules: %w", err)
	}
	resp.Body.Close()
	return nil
}
