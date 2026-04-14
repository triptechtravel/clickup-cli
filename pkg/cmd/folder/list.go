package folder

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdFolderList returns the folder list command.
func NewCmdFolderList(f *cmdutil.Factory) *cobra.Command {
	var (
		spaceID   string
		archived  bool
		jsonFlags cmdutil.JSONFlags
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List folders in a space",
		Long:  "List all folders in a ClickUp space.",
		Example: `  # List folders in your default space
  clickup folder list

  # List folders in a specific space
  clickup folder list --space 12345

  # Include archived folders
  clickup folder list --archived`,
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

			if spaceID == "" {
				dir, _ := os.Getwd()
				spaceID = cfg.SpaceForDir(dir)
			}
			if spaceID == "" {
				return fmt.Errorf("no space configured. Use --space or run 'clickup space select' first")
			}

			folders, err := apiv2.GetFoldersLocal(context.Background(), client, spaceID, archived)
			if err != nil {
				return fmt.Errorf("failed to fetch folders: %w", err)
			}

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, folders)
			}

			if len(folders) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "No folders found.")
				return nil
			}

			tp := tableprinter.New(f.IOStreams)

			for _, folder := range folders {
				tp.AddField(folder.ID)
				tp.AddField(folder.Name)
				tp.EndRow()
			}

			if err := tp.Render(); err != nil {
				return err
			}

			cs := f.IOStreams.ColorScheme()
			fmt.Fprintln(f.IOStreams.Out)
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("---"))
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("Quick actions:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup folder select\n", cs.Gray("Select:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup list list --folder <id>\n", cs.Gray("Lists:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup folder list --json\n", cs.Gray("JSON:"))

			return nil
		},
	}

	cmd.Flags().StringVar(&spaceID, "space", "", "Space ID (defaults to configured space)")
	cmd.Flags().BoolVar(&archived, "archived", false, "Include archived folders")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}
