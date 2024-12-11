package content

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Manage content sources",
		Long: `Content commands allow you to add, update, and remove content sources
that can be shown on displays. Each piece of content has a unique name
and can be configured with properties that affect how it is displayed.`,
	}

	cmd.AddCommand(
		newAddCommand(),
		newRemoveCommand(),
		newUpdateCommand(),
		newListCommand(),
	)

	return cmd
}
