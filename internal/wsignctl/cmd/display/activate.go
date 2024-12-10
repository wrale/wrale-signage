package display

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// newActivateCommand creates a command for activating displays showing setup codes
func newActivateCommand() *cobra.Command {
	var (
		siteID   string
		zone     string
		position string
		labels   []string
	)

	cmd := &cobra.Command{
		Use:   "activate CODE",
		Short: "Activate a display showing a setup code",
		Long: `Activate a display that is showing an activation code by providing its
location information and any additional properties.

The activation code should be visible on the display's screen after it has
connected to the displays.{domain} endpoint.`,
		Example: `  # Activate a display showing code BLUE-FISH
  wsignctl display activate BLUE-FISH --site-id=hq --zone=lobby --position=north
  
  # Activate with additional metadata
  wsignctl display activate CAKE-MOON --site-id=hq --zone=cafeteria --position=menu-1 \
    --label=orientation=portrait`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]

			// Parse labels into properties map
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

			// Build registration request
			reg := &v1alpha1.DisplayRegistrationRequest{
				Name: "", // Server will generate name if not provided
				Location: v1alpha1.DisplayLocation{
					SiteID:   siteID,
					Zone:     zone,
					Position: position,
				},
				ActivationCode: code,
			}

			if err := client.ActivateDisplay(cmd.Context(), reg); err != nil {
				return fmt.Errorf("error activating display: %w", err)
			}

			fmt.Printf("Display activated successfully\n")
			return nil
		},
	}

	// Add location flags
	cmd.Flags().StringVar(&siteID, "site-id", "", "Site identifier (required)")
	cmd.Flags().StringVar(&zone, "zone", "", "Zone within site (required)")
	cmd.Flags().StringVar(&position, "position", "", "Position within zone (required)")
	cmd.Flags().StringArrayVar(&labels, "label", nil, "Additional labels in key=value format")

	cmd.MarkFlagRequired("site-id")
	cmd.MarkFlagRequired("zone")
	cmd.MarkFlagRequired("position")

	return cmd
}
