// Package content implements the content management commands
package content

import (
	"github.com/spf13/cobra"
)

// NewCommand creates a new content command
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "content",
		Short: "Manage content sources",
		Long: `The content command provides subcommands for managing content sources in the system.

A content source defines where wsignd can fetch content from when redirecting displays.
Each source has a URL and optional settings that control how content is accessed.

For example:
- Menu boards from https://menu.example.com
- Company intranet at https://intranet.example.com/signage
- Emergency notifications from https://alerts.example.com`,
	}

	cmd.AddCommand(
		newAddCmd(),
		newListCmd(),
		newUpdateCmd(),
		newRemoveCmd(),
	)

	return cmd
}
