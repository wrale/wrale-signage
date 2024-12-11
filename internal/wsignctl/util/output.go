// Package util provides shared utilities for the CLI
package util

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
)

// PrintJSON writes a JSON representation of v to w with proper indentation
func PrintJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// NewTabWriter creates a new tabwriter configured for CLI output
func NewTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// FormatDuration formats a duration in a human-friendly way for CLI output
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return "Just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

// FormatProperties formats a map of properties as a comma-separated string of key=value pairs
func FormatProperties(props map[string]string) string {
	if len(props) == 0 {
		return ""
	}
	var pairs []string
	for k, v := range props {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}
