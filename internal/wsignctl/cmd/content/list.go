package content

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/internal/wsignctl/client"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

func newListCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List content sources",
		Long: `List all configured content sources.

This shows where displays can be redirected to fetch content from.`,
		Example: `  # List all content sources
  wsignctl content list
  
  # Show detailed JSON output
  wsignctl content list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.GetClient()
			if err != nil {
				return err
			}

			sources, err := c.ListContentSources(cmd.Context())
			if err != nil {
				return fmt.Errorf("error listing content sources: %w", err)
			}

			switch output {
			case "json":
				return util.PrintJSON(cmd.OutOrStdout(), sources)

			default:
				tw := util.NewTabWriter(cmd.OutOrStdout())
				defer tw.Flush()

				// Print header
				fmt.Fprintf(tw, "NAME\tURL\tTYPE\tPROPERTIES\tLAST VALIDATED\tHASH\n")

				// Print each source
				for _, s := range sources {
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
						s.Name,
						s.Spec.URL,
						s.Spec.Type,
						util.FormatProperties(s.Spec.Properties),
						s.Status.LastValidated.Format("2006-01-02 15:04:05"),
						s.Status.Hash,
					)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format (table, json)")

	return cmd
}
