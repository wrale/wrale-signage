package display

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a display",
		Long: `Remove a display from the system. This will prevent the display from
loading content until it is activated again.

This command should be used when:
- Decommissioning a display permanently
- Removing test/temporary displays
- Cleaning up stale display entries

Note that deleting a display does not affect the physical display device,
which will continue trying to connect until reactivated or reconfigured.`,
		Example: `  # Delete a display
  wsignctl display delete lobby-north
  
  # Delete a test display
  wsignctl display delete temp-display-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteDisplay(cmd.Context(), name); err != nil {
				return fmt.Errorf("error deleting display: %w", err)
			}

			fmt.Printf("Display %q deleted successfully\n", name)
			return nil
		},
	}

	return cmd
}
