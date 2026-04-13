package folder

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdFolderList returns the folder list command.
func NewCmdFolderList(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags
	var spaceID string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List folders in a space",
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

			targetSpace := spaceID
			if targetSpace == "" {
				targetSpace = cfg.Space
			}

			if targetSpace == "" {
				return fmt.Errorf("no space specified or configured. Use --space or run 'clickup space select'")
			}

			folders, _, err := client.Clickup.Folders.GetFolders(context.Background(), targetSpace, false)
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

			return tp.Render()
		},
	}

	cmd.Flags().StringVar(&spaceID, "space", "", "Space ID (defaults to configured space)")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)
	return cmd
}
