package display

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

// newActivateCommand creates a command for activating displays showing setup codes
func newActivateCommand() *cobra.Command {
	var (
		site     string
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
  wsignctl display activate BLUE-FISH --site=hq --zone=lobby --position=north
  
  # Activate with additional metadata
  wsignctl display activate CAKE-MOON --site=hq --zone=cafeteria --position=menu-1 \
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
				return fmt.Errorf("client setup failed: %w", err)
			}

			// Generate default name from site and position
			displayName := fmt.Sprintf("%s-%s-%s", site, zone, position)

			// Build registration request
			reg := &v1alpha1.DisplayRegistrationRequest{
				Name: displayName,
				Location: v1alpha1.DisplayLocation{
					SiteID:   site,
					Zone:     zone,
					Position: position,
				},
				ActivationCode: code,
			}

			// Attempt to activate the display
			display, err := client.ActivateDisplay(cmd.Context(), reg)
			if err != nil {
				if strings.Contains(err.Error(), "activation code not found") {
					fmt.Fprintf(os.Stderr, "Error: The activation code %q was not found.\n", code)
					fmt.Fprintf(os.Stderr, "Make sure:\n")
					fmt.Fprintf(os.Stderr, "  1. The display is online and showing the code\n")
					fmt.Fprintf(os.Stderr, "  2. You've entered the code exactly as shown\n")
					fmt.Fprintf(os.Stderr, "  3. The code hasn't expired (valid for 15 minutes)\n")
					return fmt.Errorf("invalid activation code")
				} else if strings.Contains(err.Error(), "display already exists") {
					fmt.Fprintf(os.Stderr, "Error: A display named %q already exists.\n", displayName)
					fmt.Fprintf(os.Stderr, "Either:\n")
					fmt.Fprintf(os.Stderr, "  1. Choose a different site/zone/position combination\n")
					fmt.Fprintf(os.Stderr, "  2. Delete the existing display first\n")
					return fmt.Errorf("display name conflict")
				} else if strings.Contains(err.Error(), "404") {
					fmt.Fprintf(os.Stderr, "Error: The display activation endpoint was not found.\n")
					fmt.Fprintf(os.Stderr, "Make sure:\n")
					fmt.Fprintf(os.Stderr, "  1. The API server URL is correct\n")
					fmt.Fprintf(os.Stderr, "  2. The server is running and reachable\n")
					return fmt.Errorf("endpoint not found")
				}
				return fmt.Errorf("display activation failed: %w", err)
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
	cmd.Flags().StringVar(&site, "site", "", "Site identifier (required)")
	cmd.Flags().StringVar(&zone, "zone", "", "Zone within site (required)")
	cmd.Flags().StringVar(&position, "position", "", "Position within zone (required)")
	cmd.Flags().StringArrayVar(&labels, "label", nil, "Additional labels in key=value format")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output format (json)")

	// Mark required flags
	requiredFlags := []string{"site", "zone", "position"}
	for _, flag := range requiredFlags {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			panic(fmt.Sprintf("failed to mark %q flag as required: %v", flag, err))
		}
	}

	return cmd
}
