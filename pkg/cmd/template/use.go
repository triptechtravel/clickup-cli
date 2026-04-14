package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	clickupv2 "github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdTemplateUse returns the template use command.
func NewCmdTemplateUse(f *cmdutil.Factory) *cobra.Command {
	var (
		listID string
		name   string
	)

	cmd := &cobra.Command{
		Use:   "use <template-id>",
		Short: "Create a task from a template",
		Long:  "Create a new task from an existing task template.",
		Example: `  # Create a task from a template
  clickup template use t-12345 --list 67890 --name "New Task from Template"`,
		Args:    cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateID := args[0]

			if listID == "" {
				return fmt.Errorf("--list is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			req := &clickupv2.CreateTaskFromTemplateJSONRequest{
				Name: name,
			}

			_, err = apiv2.CreateTaskFromTemplate(context.Background(), client, listID, templateID, req)
			if err != nil {
				return fmt.Errorf("failed to create task from template: %w", err)
			}

			cs := f.IOStreams.ColorScheme()
			fmt.Fprintf(f.IOStreams.Out, "%s Task %s created from template %s\n", cs.Green("!"), cs.Bold(name), templateID)

			return nil
		},
	}

	cmd.Flags().StringVar(&listID, "list", "", "List ID to create the task in (required)")
	cmd.Flags().StringVar(&name, "name", "", "Name for the new task (required)")

	return cmd
}
