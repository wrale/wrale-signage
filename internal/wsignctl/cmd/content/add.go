package content

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

func newAddCmd() *cobra.Command {
	var (
		url         string
		contentType string
		properties  []string
	)

	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a content source",
		Long: `Add a new content source that displays can be redirected to.

A content source needs:
- A unique name for referring to it in redirect rules
- A URL where content can be found
- A content type that identifies what kind of content this is
- Optional properties for additional metadata`,
		Example: `  # Add a basic content source
  wsignctl content add menus --url=https://menu.example.com --type=menu
  
  # Add source with additional properties
  wsignctl content add intranet \
    --url=https://intranet.example.com/signage \
    --type=internal \
    --property=department=hr \
    --property=audience=employees`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Parse properties into map
			props := make(map[string]string)
			for _, prop := range properties {
				parts := strings.SplitN(prop, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid property format %q - use Key=Value", prop)
				}
				props[parts[0]] = parts[1]
			}

			// Create the content source
			source := &v1alpha1.ContentSource{
				TypeMeta: v1alpha1.TypeMeta{
					Kind:       "ContentSource",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: v1alpha1.ObjectMeta{
					Name: name,
				},
				Spec: v1alpha1.ContentSourceSpec{
					URL:        url,
					Type:       contentType,
					Properties: props,
				},
			}

			// Get client using the command context
			c, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return fmt.Errorf("failed to create API client: %w", err)
			}

			if err := c.AddContentSource(cmd.Context(), source); err != nil {
				return fmt.Errorf("error adding content source: %w", err)
			}

			fmt.Printf("Content source %q added\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "URL where content can be found (required)")
	cmd.Flags().StringVar(&contentType, "type", "", "Type of content (required)")
	cmd.Flags().StringArrayVar(&properties, "property", nil, "Additional properties in Key=Value format")

	cmd.MarkFlagRequired("url")
	cmd.MarkFlagRequired("type")

	return cmd
}
