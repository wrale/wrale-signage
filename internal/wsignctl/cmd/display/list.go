package display

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

// newListCommand creates a command for listing and filtering displays
func newListCommand() *cobra.Command {
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

			// Format output based on requested format
			switch output {
			case "json":
				return util.PrintJSON(cmd.OutOrStdout(), displays)
			default:
				tw := util.NewTabWriter(cmd.OutOrStdout())
				defer tw.Flush()

				// Print header row
				fmt.Fprintf(tw, "NAME\tSITE\tZONE\tPOSITION\tSTATE\tLAST SEEN\tPROPERTIES\n")

				// Print each display as a row
				for _, d := range displays {
					lastSeen := util.FormatDuration(time.Since(d.Status.LastSeen))
					props := util.FormatProperties(d.Spec.Properties)

					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
						d.Name,
						d.Spec.Location.SiteID,
						d.Spec.Location.Zone,
						d.Spec.Location.Position,
						d.Status.State,
						lastSeen,
						props)
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
