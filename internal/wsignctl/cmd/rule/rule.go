// Package rule implements redirect rule management commands
package rule

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the rule management command
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage content redirect rules",
		Long: `The rule command manages content redirect rules that determine what
content displays see.

Rules are evaluated in priority order (highest first) when a display requests
content. The first matching rule determines which content URL the display
receives.

For example, you might have rules like:
1. Emergency notifications for all displays (priority 1000)
2. Cafeteria menu boards during lunch hours (priority 800)
3. Default welcome content for lobby displays (priority 500)
4. Company news for all other displays (priority 100)

Each rule includes:
- A priority number that determines evaluation order
- Location selectors that determine which displays it applies to
- Content type and version to redirect to
- Optional schedule for when the rule is active`,
	}

	// Add subcommands
	cmd.AddCommand(
		newAddCommand(),
		newListCommand(),
		newUpdateCommand(),
		newRemoveCommand(),
	)

	return cmd
}
