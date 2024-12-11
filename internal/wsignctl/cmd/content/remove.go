package content

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

func newRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove NAME",
		Short: "Remove content source",
		Example: `  # Remove content source
  wsignctl content remove welcome`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			if err := client.RemoveContent(cmd.Context(), name); err != nil {
				return fmt.Errorf("error removing content: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Content %q removed\n", name)
			return nil
		},
	}

	return cmd
}
