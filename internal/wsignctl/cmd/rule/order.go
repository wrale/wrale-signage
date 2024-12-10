package rule

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

// newOrderCmd creates a command for reordering redirect rules
func newOrderCmd() *cobra.Command {
	opts := &options{}

	cmd := &cobra.Command{
		Use:   "order NAME",
		Short: "Change rule evaluation order",
		Long: `Modify the order in which rules are evaluated.

Rules are normally evaluated in descending priority order. This command
provides an easier way to reorder rules than manually updating priorities.

You can:
- Move a rule before or after another rule
- Move a rule to the start or end of the list
- The system will automatically adjust priorities to maintain the desired order`,
		Example: `  # Move a rule before another
  wsignctl rule order menu-special --before daily-menu
  
  # Move a rule after another
  wsignctl rule order site-notice --after welcome-msg
  
  # Move to start/end
  wsignctl rule order emergency --to-start
  wsignctl rule order fallback --to-end`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Validate flags
			count := 0
			if opts.beforeRule != "" {
				count++
			}
			if opts.afterRule != "" {
				count++
			}
			if opts.moveToStart {
				count++
			}
			if opts.moveToEnd {
				count++
			}
			if count != 1 {
				return fmt.Errorf("exactly one of --before, --after, --to-start, or --to-end is required")
			}

			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			var targetRule string
			var position string

			switch {
			case opts.beforeRule != "":
				targetRule = opts.beforeRule
				position = "before"
			case opts.afterRule != "":
				targetRule = opts.afterRule
				position = "after"
			case opts.moveToStart:
				position = "start"
			case opts.moveToEnd:
				position = "end"
			}

			if err := client.ReorderRedirectRule(cmd.Context(), name, position, targetRule); err != nil {
				return fmt.Errorf("error reordering rule: %w", err)
			}

			if targetRule != "" {
				fmt.Printf("Moved rule %q %s %q\n", name, position, targetRule)
			} else {
				fmt.Printf("Moved rule %q to %s of list\n", name, position)
			}
			return nil
		},
	}

	// Add ordering flags
	f := cmd.Flags()
	f.StringVar(&opts.beforeRule, "before", "", "Place rule before this one")
	f.StringVar(&opts.afterRule, "after", "", "Place rule after this one")
	f.BoolVar(&opts.moveToStart, "to-start", false, "Move rule to start of evaluation order")
	f.BoolVar(&opts.moveToEnd, "to-end", false, "Move rule to end of evaluation order")

	return cmd
}
