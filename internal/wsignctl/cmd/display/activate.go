package display

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

// newActivateCommand creates a command for activating displays showing setup codes
func newActivateCommand() *cobra.Command {
	var (
		siteID   string
		zone     string
		position string
		labels   []string
		output   string
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

			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			// Generate default name from site and position
			displayName := fmt.Sprintf("%s-%s-%s", siteID, zone, position)

			// Build registration request
			reg := &v1alpha1.DisplayRegistrationRequest{
				Name: displayName,
				Location: v1alpha1.DisplayLocation{
					SiteID:   siteID,
					Zone:     zone,
					Position: position,
				},
				ActivationCode: code,
			}

			// Attempt to activate the display
			display, err := client.ActivateDisplay(cmd.Context(), reg)
			if err != nil {
				return fmt.Errorf("error activating display: %w", err)
			}

			// Format and display the result
			if output == "json" {
				return util.PrintJSON(cmd.OutOrStdout(), display)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Display activated successfully!\n\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Details:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  Name:     %s\n", display.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "  ID:       %s\n", display.ObjectMeta.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Location: %s/%s/%s\n",
				display.Spec.Location.SiteID,
				display.Spec.Location.Zone,
				display.Spec.Location.Position)
			fmt.Fprintf(cmd.OutOrStdout(), "  State:    %s\n", display.Status.State)

			if len(display.Spec.Properties) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\nProperties:\n")
				for k, v := range display.Spec.Properties {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", k, v)
				}
			}

			return nil
		},
	}

	// Add location flags
	cmd.Flags().StringVar(&siteID, "site-id", "", "Site identifier (required)")
	cmd.Flags().StringVar(&zone, "zone", "", "Zone within site (required)")
	cmd.Flags().StringVar(&position, "position", "", "Position within zone (required)")
	cmd.Flags().StringArrayVar(&labels, "label", nil, "Additional labels in key=value format")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output format (json)")

	// Mark required flags and handle potential errors
	requiredFlags := []string{"site-id", "zone", "position"}
	for _, flag := range requiredFlags {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			// Since this is during command construction, we should panic
			// This follows cobra's pattern for command setup errors
			panic(fmt.Sprintf("failed to mark %q flag as required: %v", flag, err))
		}
	}

	return cmd
}
