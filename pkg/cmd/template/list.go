package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// templateItem is a unified representation for display.
type templateItem struct {
	Name string
	ID   string
	Kind string
}

// NewCmdTemplateList returns the template list command.
func NewCmdTemplateList(f *cmdutil.Factory) *cobra.Command {
	var (
		kind      string
		jsonFlags cmdutil.JSONFlags
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List templates",
		Long: `List templates available in your ClickUp workspace.

Use --type to filter by template type: task (default), folder, or list.`,
		Example: `  # List task templates
  clickup template list

  # List folder templates
  clickup template list --type folder

  # List list templates
  clickup template list --type list

  # Output as JSON
  clickup template list --json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			cfg, err := f.Config()
			if err != nil {
				return err
			}

			teamID := cfg.Workspace
			if teamID == "" {
				return fmt.Errorf("no workspace configured. Run 'clickup auth' first")
			}

			ctx := context.Background()
			var items []templateItem

			switch kind {
			case "folder":
				resp, err := apiv2.GetFolderTemplates(ctx, client, teamID)
				if err != nil {
					return fmt.Errorf("failed to fetch folder templates: %w", err)
				}
				if jsonFlags.WantsJSON() {
					return jsonFlags.OutputJSON(f.IOStreams.Out, resp.Templates)
				}
				for _, t := range resp.Templates {
					name := ""
					if t.Name != nil {
						name = *t.Name
					}
					id := ""
					if t.ID != nil {
						id = *t.ID
					}
					items = append(items, templateItem{Name: name, ID: id, Kind: "folder"})
				}
			case "list":
				resp, err := apiv2.GetListTemplates(ctx, client, teamID)
				if err != nil {
					return fmt.Errorf("failed to fetch list templates: %w", err)
				}
				if jsonFlags.WantsJSON() {
					return jsonFlags.OutputJSON(f.IOStreams.Out, resp.Templates)
				}
				for _, t := range resp.Templates {
					name := ""
					if t.Name != nil {
						name = *t.Name
					}
					id := ""
					if t.ID != nil {
						id = *t.ID
					}
					items = append(items, templateItem{Name: name, ID: id, Kind: "list"})
				}
			default:
				// Task templates — note: Templates is []string.
				resp, err := apiv2.GetTaskTemplates(ctx, client, teamID)
				if err != nil {
					return fmt.Errorf("failed to fetch task templates: %w", err)
				}
				if jsonFlags.WantsJSON() {
					return jsonFlags.OutputJSON(f.IOStreams.Out, resp.Templates)
				}
				for _, t := range resp.Templates {
					items = append(items, templateItem{Name: t, ID: t, Kind: "task"})
				}
			}

			if len(items) == 0 {
				fmt.Fprintf(f.IOStreams.Out, "No %s templates found.\n", kind)
				return nil
			}

			tp := tableprinter.New(f.IOStreams)

			for _, item := range items {
				tp.AddField(item.ID)
				tp.AddField(item.Name)
				tp.AddField(item.Kind)
				tp.EndRow()
			}

			if err := tp.Render(); err != nil {
				return err
			}

			cs := f.IOStreams.ColorScheme()
			fmt.Fprintln(f.IOStreams.Out)
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("---"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup template use <id> --list <list-id> --name \"Task name\"\n", cs.Gray("Use:"))

			return nil
		},
	}

	cmd.Flags().StringVar(&kind, "type", "task", "Template type: task, folder, or list")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}
