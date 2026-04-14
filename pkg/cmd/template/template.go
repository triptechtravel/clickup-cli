package template

import (
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdTemplate returns the template parent command.
func NewCmdTemplate(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage templates",
		Long:  "List and use ClickUp templates for tasks, folders, and lists.",
	}

	cmd.AddCommand(NewCmdTemplateList(f))
	cmd.AddCommand(NewCmdTemplateUse(f))

	return cmd
}
