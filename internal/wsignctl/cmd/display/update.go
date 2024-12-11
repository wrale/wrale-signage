package display

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

func newUpdateCommand() *cobra.Command {
	var (
		siteID       string
		zone         string
		position     string
		addLabels    []string
		removeLabels []string
	)

	cmd := &cobra.Command{
		Use:   "update NAME",
		Short: "Update display configuration",
		Long: `Update a display's location or properties.
		
Location changes are useful when physically moving displays. Labels can be
added or removed to update display metadata.`,
		Example: `  # Update display location
  wsignctl display update lobby-north --site-id=hq --zone=lobby --position=south
  
  # Add and remove labels
  wsignctl display update cafe-menu-1 \
    --add-label=screen-size=55 \
    --remove-label=temporary`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Parse labels to add into properties map
			addProps := make(map[string]string)
			for _, label := range addLabels {
				parts := strings.SplitN(label, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid label format %q - use key=value", label)
				}
				addProps[parts[0]] = parts[1]
			}

			// Validate remove label format
			var removeProps []string
			for _, label := range removeLabels {
				if strings.Contains(label, "=") {
					return fmt.Errorf("use just the key name to remove labels, not %q", label)
				}
				removeProps = append(removeProps, label)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Only include location in update if any location field is set
			var location *v1alpha1.DisplayLocation
			if siteID != "" || zone != "" || position != "" {
				location = &v1alpha1.DisplayLocation{
					SiteID:   siteID,
					Zone:     zone,
					Position: position,
				}
			}

			if err := client.UpdateDisplay(cmd.Context(), name, location, addProps, removeProps); err != nil {
				return fmt.Errorf("error updating display: %w", err)
			}

			fmt.Printf("Display %q updated successfully\n", name)
			return nil
		},
	}

	// Add flags for all update options
	cmd.Flags().StringVar(&siteID, "site-id", "", "New site identifier")
	cmd.Flags().StringVar(&zone, "zone", "", "New zone")
	cmd.Flags().StringVar(&position, "position", "", "New position")
	cmd.Flags().StringArrayVar(&addLabels, "add-label", nil, "Add labels in key=value format")
	cmd.Flags().StringArrayVar(&removeLabels, "remove-label", nil, "Remove labels by key")

	return cmd
}
