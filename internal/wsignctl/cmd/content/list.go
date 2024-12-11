package content

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

func newListCommand() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List content sources",
		Example: `  # List all content sources
  wsignctl content list

  # Show content sources in JSON format
  wsignctl content list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			content, err := client.ListContent(cmd.Context())
			if err != nil {
				return fmt.Errorf("error listing content: %w", err)
			}

			if output == "json" {
				return util.PrintJSON(cmd.OutOrStdout(), content)
			}

			// Table output
			fmt.Fprintln(cmd.OutOrStdout(), "\nContent Sources:")
			fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", 80))
			fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-10s %-20s %s\n", "NAME", "TYPE", "DURATION", "PATH")
			fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", 80))

			for _, c := range content {
				fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-10s %-20s %s\n",
					c.Name,
					c.Type,
					c.Duration,
					c.Path,
				)
			}
			fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("-", 80))

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output format (json)")
	return cmd
}
