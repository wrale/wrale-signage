package rule

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

// newRemoveCmd creates a command for removing redirect rules
func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove NAME",
		Short: "Remove a redirect rule",
		Long: `Remove a redirect rule from the system.

This immediately stops the rule from being considered during content
redirects. Any displays that were showing content due to this rule
will fall through to the next matching rule.`,
		Example: `  # Remove a single rule
  wsignctl rule remove old-menu

  # Remove multiple rules
  wsignctl rule remove rule1 rule2 rule3`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			// Remove each specified rule
			for _, name := range args {
				if err := client.RemoveRedirectRule(cmd.Context(), name); err != nil {
					return fmt.Errorf("error removing rule %q: %w", name, err)
				}
				fmt.Printf("Rule %q removed\n", name)
			}

			return nil
		},
	}

	return cmd
}
