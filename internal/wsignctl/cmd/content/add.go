package content

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

type addOptions struct {
	url         string
	duration    time.Duration
	contentType string
	properties  []string
}

func newAddCommand() *cobra.Command {
	var opts addOptions

	cmd := &cobra.Command{
		Use:   "add NAME [flags]",
		Short: "Add new content source",
		Long: `Add a new content source by specifying its URL and display parameters.
Content must be accessible via HTTP/HTTPS and should be optimized for display use.`,
		Example: `  # Add content from a URL
  wsignctl content add welcome \
    --url https://content.example.com/welcome.html \
    --duration 10s

  # Add content with properties
  wsignctl content add news \
    --url https://content.example.com/news.html \
    --duration 15s \
    --property department=marketing \
    --property priority=high`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Validate URL
			if opts.url == "" {
				return fmt.Errorf("content URL is required")
			}
			if _, err := url.ParseRequestURI(opts.url); err != nil {
				return fmt.Errorf("invalid content URL: %v", err)
			}

			// Parse properties into map
			properties := make(map[string]string)
			for _, prop := range opts.properties {
				parts := strings.SplitN(prop, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid property format %q - use key=value", prop)
				}
				properties[parts[0]] = parts[1]
			}

			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			return client.CreateContent(cmd.Context(), name, opts.url, opts.duration, opts.contentType, properties)
		},
	}

	cmd.Flags().StringVar(&opts.url, "url", "", "Content URL (required)")
	cmd.Flags().DurationVar(&opts.duration, "duration", 10*time.Second, "How long to display content")
	cmd.Flags().StringVar(&opts.contentType, "type", "static-page", "Type of content")
	cmd.Flags().StringArrayVar(&opts.properties, "property", nil, "Additional properties in key=value format")

	if err := cmd.MarkFlagRequired("url"); err != nil {
		panic(fmt.Sprintf("failed to mark url flag as required: %v", err))
	}

	return cmd
}
