package content

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

type addOptions struct {
	path        string
	duration    time.Duration
	contentType string
	properties  []string
}

func newAddCommand() *cobra.Command {
	var opts addOptions

	cmd := &cobra.Command{
		Use:   "add NAME [flags]",
		Short: "Add new content source",
		Long: `Add a new content source. Content can be loaded from a local path,
during development, or from a remote URL in production.`,
		Example: `  # Add a simple test content source
  wsignctl content add welcome \
    --path demo/content/welcome.html \
    --duration 10s

  # Add content with properties
  wsignctl content add news \
    --path demo/content/news.html \
    --duration 15s \
    --property department=marketing \
    --property priority=high`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
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

			return client.CreateContent(cmd.Context(), name, opts.path, opts.duration, opts.contentType, properties)
		},
	}

	cmd.Flags().StringVar(&opts.path, "path", "", "Path to content file or directory (required)")
	cmd.Flags().DurationVar(&opts.duration, "duration", 10*time.Second, "How long to display content")
	cmd.Flags().StringVar(&opts.contentType, "type", "static-page", "Type of content")
	cmd.Flags().StringArrayVar(&opts.properties, "property", nil, "Additional properties in key=value format")

	if err := cmd.MarkFlagRequired("path"); err != nil {
		panic(fmt.Sprintf("failed to mark path flag as required: %v", err))
	}

	return cmd
}
