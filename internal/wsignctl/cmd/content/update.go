package content

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

func newUpdateCmd() *cobra.Command {
	var (
		url         string
		addProps    []string
		removeProps []string
	)

	cmd := &cobra.Command{
		Use:   "update NAME",
		Short: "Update a content source",
		Long: `Update the configuration of an existing content source.

You can modify:
- The URL where content is found
- Properties (add or remove)`,
		Example: `  # Update URL
  wsignctl content update menus --url=https://newmenu.example.com
  
  # Modify properties
  wsignctl content update weather \
    --add-property=refresh=5m \
    --remove-property=old-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Parse properties to add/update
			properties := make(map[string]string)
			for _, prop := range addProps {
				parts := strings.SplitN(prop, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid property format %q - use Key=Value", prop)
				}
				properties[parts[0]] = parts[1]
			}

			// Mark properties to remove with empty values
			for _, prop := range removeProps {
				if strings.Contains(prop, "=") {
					return fmt.Errorf("use just the property name to remove, not %q", prop)
				}
				properties[prop] = ""
			}

			// Build update
			update := &v1alpha1.ContentSourceUpdate{
				Properties: properties,
			}
			if url != "" {
				update.URL = &url
			}

			c, err := util.GetClientFromCommand(cmd)
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
	cmd.Flags().StringArrayVar(&addProps, "add-property", nil, "Add properties in Key=Value format")
	cmd.Flags().StringArrayVar(&removeProps, "remove-property", nil, "Remove properties by name")

	return cmd
}
