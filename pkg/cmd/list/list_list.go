package list

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdListList returns the list list command.
func NewCmdListList(f *cmdutil.Factory) *cobra.Command {
	var (
		folderID  string
		spaceID   string
		archived  bool
		jsonFlags cmdutil.JSONFlags
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List ClickUp lists in a folder or space",
		Long: `List ClickUp lists. If --folder is provided, lists within that folder
are returned. If only --space is provided, folderless lists in that space
are returned.`,
		Example: `  # List lists in your default folder
  clickup list list

  # List lists in a specific folder
  clickup list list --folder 12345

  # List folderless lists in a space
  clickup list list --space 67890`,
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.ApiClient()
			if err != nil {
				return err
			}

			cfg, err := f.Config()
			if err != nil {
				return err
			}

			dir, _ := os.Getwd()

			if folderID == "" {
				folderID = cfg.FolderForDir(dir)
			}
			if spaceID == "" {
				spaceID = cfg.SpaceForDir(dir)
			}

			var lists []clickup.List
			ctx := context.Background()

			if folderID != "" {
				lists, err = apiv2.GetListsLocal(ctx, client, folderID, archived)
				if err != nil {
					return fmt.Errorf("failed to fetch lists: %w", err)
				}
			} else if spaceID != "" {
				lists, err = apiv2.GetFolderlessListsLocal(ctx, client, spaceID, archived)
				if err != nil {
					return fmt.Errorf("failed to fetch folderless lists: %w", err)
				}
			} else {
				return fmt.Errorf("no folder or space configured. Use --folder, --space, or run 'clickup folder select' / 'clickup space select' first")
			}

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, lists)
			}

			if len(lists) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "No lists found.")
				return nil
			}

			tp := tableprinter.New(f.IOStreams)

			for _, l := range lists {
				tp.AddField(l.ID)
				tp.AddField(l.Name)
				tp.EndRow()
			}

			if err := tp.Render(); err != nil {
				return err
			}

			cs := f.IOStreams.ColorScheme()
			fmt.Fprintln(f.IOStreams.Out)
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("---"))
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("Quick actions:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup list select\n", cs.Gray("Select:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup task list --list-id <id>\n", cs.Gray("Tasks:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup list list --json\n", cs.Gray("JSON:"))

			return nil
		},
	}

	cmd.Flags().StringVar(&folderID, "folder", "", "Folder ID (defaults to configured folder)")
	cmd.Flags().StringVar(&spaceID, "space", "", "Space ID (defaults to configured space, used for folderless lists)")
	cmd.Flags().BoolVar(&archived, "archived", false, "Include archived lists")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}
