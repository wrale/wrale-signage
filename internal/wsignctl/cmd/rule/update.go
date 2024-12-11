package rule

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/api/types/v1alpha1"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

// newUpdateCmd creates a command for updating existing redirect rules
func newUpdateCmd() *cobra.Command {
	opts := &options{
		priority: new(int),
	}

	cmd := &cobra.Command{
		Use:   "update NAME",
		Short: "Update an existing redirect rule",
		Long: `Update properties of an existing redirect rule.

You can modify:
- Rule priority
- Location selectors
- Content target
- Schedule constraints

The rule name cannot be changed. Create a new rule with the desired
name and remove the old one if you need to rename a rule.`,
		Example: `  # Update priority
  wsignctl rule update lobby-welcome --priority 600

  # Change content target
  wsignctl rule update menu-board \
    --content-type=menu \
    --version=spring-2024 \
    --hash=abc123

  # Modify schedule
  wsignctl rule update daily-special \
    --days=Mon,Tue,Wed,Thu,Fri \
    --time=11:00-14:00`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Parse schedule if any schedule flags were set
			var schedule *v1alpha1.Schedule
			if cmd.Flags().Changed("start") || cmd.Flags().Changed("end") ||
				cmd.Flags().Changed("days") || cmd.Flags().Changed("time") {
				var err error
				schedule, err = util.ParseSchedule(
					opts.startTime,
					opts.endTime,
					opts.daysOfWeek,
					opts.timeOfDay,
				)
				if err != nil {
					return fmt.Errorf("invalid schedule: %w", err)
				}
			}

			// Build the update
			update := &v1alpha1.RedirectRuleUpdate{}

			// Only include changed fields
			if cmd.Flags().Changed("priority") {
				update.Priority = opts.priority
			}
			if cmd.Flags().Changed("site-id") || cmd.Flags().Changed("zone") || cmd.Flags().Changed("position") {
				update.DisplaySelector = &v1alpha1.DisplaySelector{
					SiteID:   opts.siteID,
					Zone:     opts.zone,
					Position: opts.position,
				}
			}
			if cmd.Flags().Changed("content-type") || cmd.Flags().Changed("version") || cmd.Flags().Changed("hash") {
				update.Content = &v1alpha1.ContentRedirect{
					ContentType: opts.contentType,
					Version:     opts.version,
					Hash:        opts.hash,
				}
			}
			if schedule != nil {
				update.Schedule = schedule
			}

			// Update through API
			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			if err := client.UpdateRedirectRule(cmd.Context(), name, update); err != nil {
				return fmt.Errorf("error updating rule: %w", err)
			}

			fmt.Printf("Rule %q updated\n", name)
			return nil
		},
	}

	// Add flags for updatable fields
	f := cmd.Flags()
	f.IntVar(opts.priority, "priority", 0, "Rule priority (higher numbers evaluated first)")
	f.StringVar(&opts.siteID, "site-id", "", "Site ID selector")
	f.StringVar(&opts.zone, "zone", "", "Zone selector")
	f.StringVar(&opts.position, "position", "", "Position selector")
	f.StringVar(&opts.contentType, "content-type", "", "Content type to redirect to")
	f.StringVar(&opts.version, "version", "", "Content version")
	f.StringVar(&opts.hash, "hash", "", "Content hash")

	// Add schedule flags
	f.StringVar(&opts.startTime, "start", "", "Rule start time (RFC3339)")
	f.StringVar(&opts.endTime, "end", "", "Rule end time (RFC3339)")
	f.StringSliceVar(&opts.daysOfWeek, "days", nil, "Active days of week (e.g., Mon,Wed,Fri)")
	f.StringVar(&opts.timeOfDay, "time", "", "Active time range (HH:MM-HH:MM)")

	return cmd
}
