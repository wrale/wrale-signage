package content

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wrale/wrale-signage/internal/wsignctl/util"
)

type updateOptions struct {
	path        string
	duration    time.Duration
	contentType string
	properties  []string
	remove      []string
}

func newUpdateCommand() *cobra.Command {
	var opts updateOptions

	cmd := &cobra.Command{
		Use:   "update NAME [flags]",
		Short: "Update content source",
		Example: `  # Update content path
  wsignctl content update welcome \
    --path demo/content/new-welcome.html

  # Update properties
  wsignctl content update news \
    --property department=sales \
    --property priority=low \
    --remove-property author`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			client, err := util.GetClientFromCommand(cmd)
			if err != nil {
				return err
			}

			// Parse properties
			properties := make(map[string]string)
			for _, prop := range opts.properties {
				parts := strings.SplitN(prop, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid property format %q - use key=value", prop)
				}
				properties[parts[0]] = parts[1]
			}

			if err := client.UpdateContent(cmd.Context(), name, opts.path, opts.duration, opts.contentType, properties, opts.remove); err != nil {
				return fmt.Errorf("error updating content: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Content %q updated\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.path, "path", "", "New content path")
	cmd.Flags().DurationVar(&opts.duration, "duration", 0, "New display duration")
	cmd.Flags().StringVar(&opts.contentType, "type", "", "New content type")
	cmd.Flags().StringArrayVar(&opts.properties, "property", nil, "Properties to add/update")
	cmd.Flags().StringArrayVar(&opts.remove, "remove-property", nil, "Properties to remove")

	return cmd
}
