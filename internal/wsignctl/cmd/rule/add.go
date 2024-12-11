package rule

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

// newAddCmd creates a command for adding new redirect rules
func newAddCmd() *cobra.Command {
	opts := &options{
		priority: new(int), // Default priority will be set in preRun
	}

	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a new redirect rule",
		Long: `Add a new rule that determines what content displays should show.

Required fields:
- NAME: A unique identifier for the rule (e.g., "lobby-welcome")
- Content type, version, and hash specifying what to show

Optional fields:
- Priority number (defaults to 500, higher numbers evaluated first)
- Location selectors to target specific displays
- Schedule constraints for time-based content

The rule's location selectors determine which displays it applies to.
Rules are evaluated in priority order until a matching rule is found.`,
		Example: `  # Basic rule for lobby displays
  wsignctl rule add lobby-welcome \
    --priority 500 \
    --zone=lobby \
    --content-type=welcome \
    --version=current \
    --hash=abc123

  # Scheduled menu board rule
  wsignctl rule add lunch-menu \
    --priority 800 \
    --zone=cafeteria \
    --content-type=menu \
    --version=current \
    --hash=def456 \
    --days=Mon,Tue,Wed,Thu,Fri \
    --time=11:00-14:00

  # Emergency notification rule
  wsignctl rule add emergency \
    --priority 1000 \
    --content-type=alert \
    --version=current \
    --hash=xyz789`,
		Args: cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			// Set default priority if not specified
			if !cmd.Flags().Changed("priority") {
				*opts.priority = 500
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Parse schedule if any schedule flags were set
			schedule, err := util.ParseSchedule(
				opts.startTime,
				opts.endTime,
				opts.daysOfWeek,
				opts.timeOfDay,
			)
			if err != nil {
				return fmt.Errorf("invalid schedule: %w", err)
			}

			// Build the rule
			rule := &v1alpha1.RedirectRule{
				Name:     name,
				Priority: *opts.priority,
				DisplaySelector: v1alpha1.DisplaySelector{
					SiteID:   opts.siteID,
					Zone:     opts.zone,
					Position: opts.position,
				},
				Content: v1alpha1.ContentRedirect{
					ContentType: opts.contentType,
					Version:     opts.version,
					Hash:        opts.hash,
				},
				Schedule: schedule,
			}

			// Add the rule through the API
			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			if err := client.AddRedirectRule(cmd.Context(), rule); err != nil {
				return fmt.Errorf("error adding rule: %w", err)
			}

			fmt.Printf("Rule %q added\n", name)
			return nil
		},
	}

	// Add flags for rule properties
	f := cmd.Flags()
	f.IntVar(opts.priority, "priority", 500, "Rule priority (higher numbers evaluated first)")
	f.StringVar(&opts.siteID, "site-id", "", "Site ID selector")
	f.StringVar(&opts.zone, "zone", "", "Zone selector")
	f.StringVar(&opts.position, "position", "", "Position selector")
	f.StringVar(&opts.contentType, "content-type", "", "Content type to redirect to (required)")
	f.StringVar(&opts.version, "version", "", "Content version (required)")
	f.StringVar(&opts.hash, "hash", "", "Content hash (required)")

	// Add schedule flags
	f.StringVar(&opts.startTime, "start", "", "Rule start time (RFC3339)")
	f.StringVar(&opts.endTime, "end", "", "Rule end time (RFC3339)")
	f.StringSliceVar(&opts.daysOfWeek, "days", nil, "Active days of week (e.g., Mon,Wed,Fri)")
	f.StringVar(&opts.timeOfDay, "time", "", "Active time range (HH:MM-HH:MM)")

	// Mark required flags and handle potential errors
	for _, flagName := range []string{"content-type", "version", "hash"} {
		if err := cmd.MarkFlagRequired(flagName); err != nil {
			// This would only happen if we specified a flag name that doesn't exist
			panic(fmt.Sprintf("failed to mark required flag %q: %v", flagName, err))
		}
	}

	return cmd
}
