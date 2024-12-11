package client

import (
	"context"
	"encoding/json"
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = fmt.Errorf("error closing response body: %v (original error: %w)", cerr, err)
		}
	}()
	return nil
}

// ListRedirectRules retrieves redirect rules matching the filter
func (c *Client) ListRedirectRules(ctx context.Context, filter *v1alpha1.RuleFilter) ([]v1alpha1.RedirectRule, error) {
	// Build query parameters for filtering
	query := make(map[string]string)
	if filter != nil {
		if filter.SiteID != "" {
			query["siteId"] = filter.SiteID
		}
		if filter.Zone != "" {
			query["zone"] = filter.Zone
		}
		if filter.Position != "" {
			query["position"] = filter.Position
		}
	}

	path := "/api/v1alpha1/rules"
	if len(query) > 0 {
		values := make([]string, 0, len(query))
		for k, v := range query {
			values = append(values, fmt.Sprintf("%s=%s", k, v))
		}
		path += "?" + values[0]
		for _, v := range values[1:] {
			path += "&" + v
		}
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list redirect rules: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = fmt.Errorf("error closing response body: %v (original error: %w)", cerr, err)
		}
	}()

	var rules []v1alpha1.RedirectRule
	if err := json.NewDecoder(resp.Body).Decode(&rules); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return rules, nil
}

// UpdateRedirectRule updates properties of an existing redirect rule
func (c *Client) UpdateRedirectRule(ctx context.Context, name string, update *v1alpha1.RedirectRuleUpdate) error {
	resp, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/api/v1alpha1/rules/%s", name), update)
	if err != nil {
		return fmt.Errorf("failed to update redirect rule: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = fmt.Errorf("error closing response body: %v (original error: %w)", cerr, err)
		}
	}()
	return nil
}

// RemoveRedirectRule deletes a redirect rule
func (c *Client) RemoveRedirectRule(ctx context.Context, name string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1alpha1/rules/%s", name), nil)
	if err != nil {
		return fmt.Errorf("failed to delete redirect rule: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = fmt.Errorf("error closing response body: %v (original error: %w)", cerr, err)
		}
	}()
	return nil
}

// ReorderRedirectRule updates the evaluation order of a rule
func (c *Client) ReorderRedirectRule(ctx context.Context, name, position, relativeTo string) error {
	update := &v1alpha1.RuleOrderUpdate{
		Position:   position,
		RelativeTo: relativeTo,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1alpha1/rules/%s/reorder", name), update)
	if err != nil {
		return fmt.Errorf("failed to reorder redirect rule: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = fmt.Errorf("error closing response body: %v (original error: %w)", cerr, err)
		}
	}()
	return nil
}
