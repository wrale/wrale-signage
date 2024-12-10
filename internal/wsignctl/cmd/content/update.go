package content

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignctl/client"
)

func newUpdateCmd() *cobra.Command {
	var (
		url         string
		contentType string
		addProps    []string
		removeProps []string
	)

	cmd := &cobra.Command{
		Use:   "update NAME",
		Short: "Update a content source",
		Long: `Update the configuration of an existing content source.

You can modify:
- The URL where content is found
- The content type
- Properties (add or remove)`,
		Example: `  # Update URL
  wsignctl content update menus --url=https://newmenu.example.com
  
  # Change content type
  wsignctl content update intranet --type=emergency
  
  # Modify properties
  wsignctl content update weather \
    --add-property=refresh=5m \
    --remove-property=old-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Parse properties to add
			addPropMap := make(map[string]string)
			for _, prop := range addProps {
				parts := strings.SplitN(prop, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid property format %q - use Key=Value", prop)
				}
				addPropMap[parts[0]] = parts[1]
			}

			// Validate properties to remove
			for _, prop := range removeProps {
				if strings.Contains(prop, "=") {
					return fmt.Errorf("use just the property name to remove, not %q", prop)
				}
			}

			// Build update
			update := &v1alpha1.ContentSourceUpdate{
				URL:              &url,
				Type:             &contentType,
				AddProperties:    addPropMap,
				RemoveProperties: removeProps,
			}

			c, err := client.GetClient()
			if err != nil {
				return err
			}

			if err := c.UpdateContentSource(cmd.Context(), name, update); err != nil {
				return fmt.Errorf("error updating content source: %w", err)
			}

			fmt.Printf("Content source %q updated\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "New URL for content")
	cmd.Flags().StringVar(&contentType, "type", "", "New content type")
	cmd.Flags().StringArrayVar(&addProps, "add-property", nil, "Add properties in Key=Value format")
	cmd.Flags().StringArrayVar(&removeProps, "remove-property", nil, "Remove properties by name")

	return cmd
}
