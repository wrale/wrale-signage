// Package display implements the display management commands
package display

import (
	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/internal/wsignctl/client"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

// NewCommand creates the display management command and its subcommands
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "display",
		Short: "Manage displays",
		Long: `The display command provides subcommands for managing displays in the system.
		
This includes creating pre-configured displays, activating displays that show setup
codes, managing display locations, and viewing display status information.`,
	}

	// Add all display-related subcommands
	cmd.AddCommand(
		newCreateCommand(),
		newActivateCommand(),
		newListCommand(),
		newUpdateCommand(),
		newDeleteCommand(),
	)

	return cmd
}

// getClient returns a configured API client
func getClient() (*client.Client, error) {
	return util.GetClient()
}
