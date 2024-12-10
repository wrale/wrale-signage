// Package cmd implements the Wrale Signage CLI commands
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
)

// newDisplayCmd creates the display management command
func newDisplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "display",
		Short: "Manage displays",
		Long: `The display command provides subcommands for managing displays in the system.
		
This includes creating pre-configured displays, activating displays that show setup
codes, managing display locations, and viewing display status information.`,
	}

	cmd.AddCommand(
		newDisplayCreateCmd(),
		newDisplayActivateCmd(),
		newDisplayListCmd(),
		newDisplayUpdateCmd(),
		newDisplayDeleteCmd(),
	)

	return cmd
}

// newDisplayCreateCmd creates a command for pre-configuring displays
func newDisplayCreateCmd() *cobra.Command {
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

			// Parse label key-value pairs
			properties := make(map[string]string)
			for _, label := range labels {
				parts := strings.SplitN(label, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid label format %q - use key=value", label)
				}
				properties[parts[0]] = parts[1]
			}

			// Create the display through our API client
			client, err := getClient()
			if err != nil {
				return err
			}

			display := &v1alpha1.DisplayStatus{
				Location: v1alpha1.DisplayLocation{
					SiteID:   siteID,
					Zone:     zone,
					Position: position,
				},
				Properties: properties,
			}

			if err := client.CreateDisplay(cmd.Context(), name, display); err != nil {
				return fmt.Errorf("error creating display: %w", err)
			}

			fmt.Printf("Display %q created\n", name)
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

// newDisplayActivateCmd creates a command for activating displays showing setup codes
func newDisplayActivateCmd() *cobra.Command {
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

			// Parse labels
			properties := make(map[string]string)
			for _, label := range labels {
				parts := strings.SplitN(label, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid label format %q - use key=value", label)
				}
				properties[parts[0]] = parts[1]
			}

			// Activate through API
			client, err := getClient()
			if err != nil {
				return err
			}

			reg := &v1alpha1.DisplayRegistration{
				ActivationCode: code,
				Location: v1alpha1.DisplayLocation{
					SiteID:   siteID,
					Zone:     zone,
					Position: position,
				},
				Properties: properties,
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

// newDisplayListCmd creates a command for listing and filtering displays
func newDisplayListCmd() *cobra.Command {
	var (
		siteID   string
		zone     string
		position string
		output   string
		showLast bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List displays",
		Long: `List displays in the system, optionally filtered by location.
		
The output can be formatted as a table (default) or as JSON for scripting.
Use --show-last to include the last content URL each display loaded.`,
		Example: `  # List all displays
  wsignctl display list
  
  # List displays at a specific site
  wsignctl display list --site-id=hq
  
  # Show display status with content information
  wsignctl display list --show-last -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			// Build location filter
			filter := v1alpha1.DisplaySelector{
				SiteID:   siteID,
				Zone:     zone,
				Position: position,
			}

			displays, err := client.ListDisplays(cmd.Context(), filter)
			if err != nil {
				return fmt.Errorf("error listing displays: %w", err)
			}

			// Format output
			switch output {
			case "json":
				return printJSON(cmd.OutOrStdout(), displays)
			default:
				tw := newTabWriter(cmd.OutOrStdout())
				defer tw.Flush()

				// Print header
				fmt.Fprintf(tw, "NAME\tSITE\tZONE\tPOSITION\tLAST SEEN\t")
				if showLast {
					fmt.Fprintf(tw, "CURRENT CONTENT\t")
				}
				fmt.Fprintf(tw, "PROPERTIES\n")

				// Print each display
				for _, d := range displays {
					lastSeen := "Never"
					if d.LastSeen != nil {
						lastSeen = formatDuration(time.Since(*d.LastSeen))
					}

					content := ""
					if showLast && d.CurrentContent != nil {
						content = d.CurrentContent.ContentURL
					}

					props := formatProperties(d.Properties)

					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t",
						d.GUID,
						d.Location.SiteID,
						d.Location.Zone,
						d.Location.Position,
						lastSeen)

					if showLast {
						fmt.Fprintf(tw, "%s\t", content)
					}

					fmt.Fprintf(tw, "%s\n", props)
				}
			}

			return nil
		},
	}

	// Add filter flags
	cmd.Flags().StringVar(&siteID, "site-id", "", "Filter by site")
	cmd.Flags().StringVar(&zone, "zone", "", "Filter by zone")
	cmd.Flags().StringVar(&position, "position", "", "Filter by position")
	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format (table, json)")
	cmd.Flags().BoolVar(&showLast, "show-last", false, "Show last content loaded")

	return cmd
}

// newDisplayUpdateCmd creates a command for updating display configuration
func newDisplayUpdateCmd() *cobra.Command {
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

			// Parse label changes
			addProps := make(map[string]string)
			for _, label := range addLabels {
				parts := strings.SplitN(label, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid label format %q - use key=value", label)
				}
				addProps[parts[0]] = parts[1]
			}

			var removeProps []string
			for _, label := range removeLabels {
				if strings.Contains(label, "=") {
					return fmt.Errorf("use just the key name to remove labels, not %q", label)
				}
				removeProps = append(removeProps, label)
			}

			// Update through API
			client, err := getClient()
			if err != nil {
				return err
			}

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

			fmt.Printf("Display %q updated\n", name)
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

// newDisplayDeleteCmd creates a command for removing displays
func newDisplayDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a display",
		Long: `Remove a display from the system. This will prevent the display from
loading content until it is activated again.`,
		Example: `  # Delete a display
  wsignctl display delete lobby-north`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteDisplay(cmd.Context(), name); err != nil {
				return fmt.Errorf("error deleting display: %w", err)
			}

			fmt.Printf("Display %q deleted\n", name)
			return nil
		},
	}
}

// Helper functions

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "Just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func formatProperties(props map[string]string) string {
	if len(props) == 0 {
		return ""
	}
	var pairs []string
	for k, v := range props {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}
