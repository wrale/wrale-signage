// Package rule implements commands for managing content redirect rules
package rule

import (
	"github.com/spf13/cobra"
)

// NewCommand creates the rule management command group
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage content redirect rules",
		Long: `The rule command manages content redirect rules that determine what content
displays see when they request content from wsignd.

Rules are evaluated in priority order (highest first) when a display makes
a request. The first matching rule determines which content URL the display
receives as a redirect.

Example rule priority hierarchy:
1. Emergency notifications (priority 1000)
2. Scheduled content like menus (priority 800)
3. Location-specific content (priority 500)
4. Default fallback content (priority 100)

Rules combine location selectors, schedules, and content targets to create
a flexible content distribution system.`,
	}

	// Add subcommands in priority order
	cmd.AddCommand(
		newAddCmd(),    // Create new rules
		newUpdateCmd(), // Modify existing rules
		newRemoveCmd(), // Delete rules
		newListCmd(),   // View current rules
		newOrderCmd(),  // Change rule priorities
	)

	return cmd
}
