package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// decodeResponse decodes a JSON response into the provided target
func decodeResponse(resp *http.Response, target interface{}) error {
	if err := handleResponse(resp); err != nil {
		return err
	}
	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("error decoding response: %w", err)
		}
	}
	return nil
}

// handleResponse processes an API response and returns an error if the status code indicates failure
func handleResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	defer resp.Body.Close()
	var apiErr struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return fmt.Errorf("HTTP %d: unable to decode error response", resp.StatusCode)
	}

	if apiErr.Message == "" {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiErr.Message)
}
