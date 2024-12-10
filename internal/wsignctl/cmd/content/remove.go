package content

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

func newRemoveCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove NAME",
		Short: "Remove a content source",
		Long: `Remove a content source from the system.

By default, this will fail if any redirect rules reference the source.
Use --force to remove it anyway and invalidate those rules.`,
		Example: `  # Remove an unused content source
  wsignctl content remove old-menus
  
  # Force remove even if referenced
  wsignctl content remove old-menus --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			c, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			if err := c.RemoveContentSource(cmd.Context(), name, force); err != nil {
				return fmt.Errorf("error removing content source: %w", err)
			}

			fmt.Printf("Content source %q removed\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Remove even if referenced by rules")

	return cmd
}
