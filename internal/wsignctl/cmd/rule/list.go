package rule

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

// newListCmd creates a command for listing redirect rules
func newListCmd() *cobra.Command {
	opts := &options{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List redirect rules",
		Long: `List all configured redirect rules in priority order.

The output shows:
- Rule priority and name
- Location selectors that determine which displays match
- Content type, version, and hash to redirect to
- Schedule constraints if the rule is time-based

Rules are shown in evaluation order (highest priority first), which
is the order they will be checked when a display requests content.`,
		Example: `  # List all rules in table format
  wsignctl rule list

  # Show detailed JSON output
  wsignctl rule list -o json

  # Filter rules by location
  wsignctl rule list --site-id=hq --zone=lobby`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			// Get rules with optional location filtering
			filter := &v1alpha1.RuleFilter{
				SiteID:   opts.siteID,
				Zone:     opts.zone,
				Position: opts.position,
			}

			rules, err := client.ListRedirectRules(cmd.Context())
			if err != nil {
				return fmt.Errorf("error listing rules: %w", err)
			}

			switch opts.output {
			case "json":
				return util.PrintJSON(cmd.OutOrStdout(), rules)

			default:
				tw := util.NewTabWriter(cmd.OutOrStdout())
				defer tw.Flush()

				// Print header
				fmt.Fprintf(tw, "PRIORITY\tNAME\tSELECTORS\tCONTENT\tSCHEDULE\n")

				// Print each rule in priority order
				for _, r := range rules {
					// Format location selectors
					selectors := util.FormatSelectors(r.DisplaySelector)

					// Format content target
					content := fmt.Sprintf("%s/%s/%s",
						r.Content.ContentType,
						r.Content.Version,
						r.Content.Hash[:8]) // Show first 8 chars of hash

					// Format schedule if present
					schedule := util.FormatSchedule(r.Schedule)

					fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\n",
						r.Priority,
						r.Name,
						selectors,
						content,
						schedule,
					)
				}
			}

			return nil
		},
	}

	// Add filter and output flags
	f := cmd.Flags()
	f.StringVar(&opts.siteID, "site-id", "", "Filter by site ID")
	f.StringVar(&opts.zone, "zone", "", "Filter by zone")
	f.StringVar(&opts.position, "position", "", "Filter by position")
	f.StringVarP(&opts.output, "output", "o", "table", "Output format (table, json)")

	return cmd
}
