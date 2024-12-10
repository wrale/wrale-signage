// Package cmd implements the Wrale Signage CLI commands
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

func newContentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Manage content sources",
		Long: `The content command provides subcommands for managing content sources in the system.

A content source defines where wsignd can fetch content from when redirecting displays.
Each source has a base URL and optional settings that control how content is accessed.

For example, you might define:
- A menu board system at https://menu.example.com
- A company intranet at https://intranet.example.com/signage
- An emergency notification system at https://alerts.example.com`,
	}

	cmd.AddCommand(
		newContentAddCmd(),
		newContentListCmd(),
		newContentUpdateCmd(),
		newContentRemoveCmd(),
	)

	return cmd
}

func newContentAddCmd() *cobra.Command {
	var (
		baseURL         string
		allowedPaths    []string
		headers         []string
		refreshInterval string
	)

	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a content source",
		Long: `Add a new content source that displays can be redirected to.

A content source needs:
- A unique name for referring to it in redirect rules
- A base URL where content can be found
- Optional allowed paths to restrict which content can be accessed
- Optional HTTP headers to include when fetching content
- Optional refresh interval for how often displays should check for updates`,
		Example: `  # Add a basic content source
  wsignctl content add menus --base-url=https://menu.example.com
  
  # Add source with path restrictions and refresh interval
  wsignctl content add intranet \
    --base-url=https://intranet.example.com/signage \
    --allowed-path=/news \
    --allowed-path=/announcements \
    --refresh=5m
    
  # Add source with custom headers
  wsignctl content add weather \
    --base-url=https://weather.example.com \
    --header="X-API-Key=secret"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Parse refresh interval if provided
			var refresh time.Duration
			if refreshInterval != "" {
				var err error
				refresh, err = time.ParseDuration(refreshInterval)
				if err != nil {
					return fmt.Errorf("invalid refresh interval %q: %w", refreshInterval, err)
				}
			}

			// Parse headers into map
			headerMap := make(map[string]string)
			for _, header := range headers {
				parts := strings.SplitN(header, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header format %q - use Key=Value", header)
				}
				headerMap[parts[0]] = parts[1]
			}

			// Create the content source
			source := &v1alpha1.ContentSource{
				Name:            name,
				BaseURL:         baseURL,
				AllowedPaths:    allowedPaths,
				Headers:         headerMap,
				RefreshInterval: refresh,
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.AddContentSource(cmd.Context(), source); err != nil {
				return fmt.Errorf("error adding content source: %w", err)
			}

			fmt.Printf("Content source %q added\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL for content (required)")
	cmd.Flags().StringArrayVar(&allowedPaths, "allowed-path", nil, "Allowed content paths")
	cmd.Flags().StringArrayVar(&headers, "header", nil, "HTTP headers to include")
	cmd.Flags().StringVar(&refreshInterval, "refresh", "", "How often displays check for updates")

	cmd.MarkFlagRequired("base-url")

	return cmd
}

func newContentListCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List content sources",
		Long: `List all configured content sources.

This shows where displays can be redirected to fetch content from.`,
		Example: `  # List all content sources
  wsignctl content list
  
  # Show detailed JSON output
  wsignctl content list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			sources, err := client.ListContentSources(cmd.Context())
			if err != nil {
				return fmt.Errorf("error listing content sources: %w", err)
			}

			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), sources)

			default:
				tw := newTabWriter(cmd.OutOrStdout())
				defer tw.Flush()

				// Print header
				fmt.Fprintf(tw, "NAME\tBASE URL\tALLOWED PATHS\tREFRESH\tHEADERS\n")

				// Print each source
				for _, s := range sources {
					paths := strings.Join(s.AllowedPaths, ",")
					if paths == "" {
						paths = "*"
					}

					refresh := "Never"
					if s.RefreshInterval > 0 {
						refresh = s.RefreshInterval.String()
					}

					headers := formatHeaders(s.Headers)

					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
						s.Name,
						s.BaseURL,
						paths,
						refresh,
						headers,
					)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format (table, json)")

	return cmd
}

func newContentUpdateCmd() *cobra.Command {
	var (
		baseURL         string
		allowedPaths    []string
		addHeaders      []string
		removeHeaders   []string
		refreshInterval string
	)

	cmd := &cobra.Command{
		Use:   "update NAME",
		Short: "Update a content source",
		Long: `Update the configuration of an existing content source.

You can modify:
- The base URL
- Allowed content paths
- HTTP headers
- Content refresh interval`,
		Example: `  # Update base URL
  wsignctl content update menus --base-url=https://newmenu.example.com
  
  # Modify allowed paths
  wsignctl content update intranet \
    --allowed-path=/news \
    --allowed-path=/announcements
    
  # Update headers
  wsignctl content update weather \
    --add-header="X-API-Key=newsecret" \
    --remove-header="X-Old-Key"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Parse refresh interval if provided
			var refresh *time.Duration
			if refreshInterval != "" {
				duration, err := time.ParseDuration(refreshInterval)
				if err != nil {
					return fmt.Errorf("invalid refresh interval %q: %w", refreshInterval, err)
				}
				refresh = &duration
			}

			// Parse headers to add
			addHeaderMap := make(map[string]string)
			for _, header := range addHeaders {
				parts := strings.SplitN(header, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header format %q - use Key=Value", header)
				}
				addHeaderMap[parts[0]] = parts[1]
			}

			// Validate headers to remove
			for _, header := range removeHeaders {
				if strings.Contains(header, "=") {
					return fmt.Errorf("use just the header name to remove, not %q", header)
				}
			}

			// Get client
			client, err := getClient()
			if err != nil {
				return err
			}

			// Build update
			update := &v1alpha1.ContentSourceUpdate{
				BaseURL:         baseURL,
				AllowedPaths:    allowedPaths,
				AddHeaders:      addHeaderMap,
				RemoveHeaders:   removeHeaders,
				RefreshInterval: refresh,
			}

			if err := client.UpdateContentSource(cmd.Context(), name, update); err != nil {
				return fmt.Errorf("error updating content source: %w", err)
			}

			fmt.Printf("Content source %q updated\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&baseURL, "base-url", "", "New base URL")
	cmd.Flags().StringArrayVar(&allowedPaths, "allowed-path", nil, "New allowed paths (replaces existing)")
	cmd.Flags().StringArrayVar(&addHeaders, "add-header", nil, "Add headers in Key=Value format")
	cmd.Flags().StringArrayVar(&removeHeaders, "remove-header", nil, "Remove headers by name")
	cmd.Flags().StringVar(&refreshInterval, "refresh", "", "New refresh interval")

	return cmd
}

func newContentRemoveCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove NAME",
		Short: "Remove a content source",
		Long: `Remove a content source from the system.

By default, this will fail if any redirect rules reference the source.
Use --force to remove it anyway and invalidate those rules.`,
		Example: `  # Remove an unused content source
  wsignctl content remove old-menus
  
  # Force remove even if referenced
  wsignctl content remove old-menus --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.RemoveContentSource(cmd.Context(), name, force); err != nil {
				return fmt.Errorf("error removing content source: %w", err)
			}

			fmt.Printf("Content source %q removed\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Remove even if referenced by rules")

	return cmd
}

// formatHeaders formats HTTP headers for display
func formatHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}

	var pairs []string
	for k, v := range headers {
		// Mask potentially sensitive header values
		if strings.Contains(strings.ToLower(k), "key") ||
			strings.Contains(strings.ToLower(k), "token") ||
			strings.Contains(strings.ToLower(k), "secret") {
			v = "********"
		}
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}
