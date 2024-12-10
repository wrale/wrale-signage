package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// newCreateCommand creates a command for pre-configuring displays
func newCreateCommand() *cobra.Command {
	var (
		siteID   string
		zone     string
		position string
		labels   []string
	)

	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Pre-configure a display",
		Long: `Create a new display entry with a known location before the display
is physically installed. The display can be activated later when it's online.

The NAME should be a human-readable identifier that helps operators locate
the display, like "lobby-north" or "cafeteria-menu-1".`,
		Example: `  # Create a display for the north lobby entrance
  wsignctl display create lobby-north --site-id=hq --zone=lobby --position=north
  
  # Create a display with additional metadata
  wsignctl display create cafe-menu-1 --site-id=hq --zone=cafeteria --position=menu-1 \
    --label=orientation=portrait --label=screen-size=55`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Parse label key-value pairs into properties map
			properties := make(map[string]string)
			for _, label := range labels {
				parts := strings.SplitN(label, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid label format %q - use key=value", label)
				}
				properties[parts[0]] = parts[1]
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Build the display object with all required fields
			display := &v1alpha1.Display{
				TypeMeta: v1alpha1.TypeMeta{
					Kind:       "Display",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: v1alpha1.ObjectMeta{
					ID:        uuid.New(),
					Name:      name,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Spec: v1alpha1.DisplaySpec{
					Location: v1alpha1.DisplayLocation{
						SiteID:   siteID,
						Zone:     zone,
						Position: position,
					},
					Properties: properties,
				},
				Status: v1alpha1.DisplayStatus{
					State:    v1alpha1.DisplayStateUnregistered,
					LastSeen: time.Now(),
					Version:  1,
				},
			}

			if err := client.CreateDisplay(cmd.Context(), name, display); err != nil {
				return fmt.Errorf("error creating display: %w", err)
			}

			fmt.Printf("Display %q created successfully\n", name)
			return nil
		},
	}

	// Add flags for all display properties
	cmd.Flags().StringVar(&siteID, "site-id", "", "Site identifier (required)")
	cmd.Flags().StringVar(&zone, "zone", "", "Zone within site (required)")
	cmd.Flags().StringVar(&position, "position", "", "Position within zone (required)")
	cmd.Flags().StringArrayVar(&labels, "label", nil, "Additional labels in key=value format")

	cmd.MarkFlagRequired("site-id")
	cmd.MarkFlagRequired("zone")
	cmd.MarkFlagRequired("position")

	return cmd
}
