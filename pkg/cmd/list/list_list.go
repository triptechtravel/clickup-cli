package list

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
	"github.com/raksul/go-clickup/clickup"
)

// NewCmdListList returns the list list command.
func NewCmdListList(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags
	var spaceID string
	var folderID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List lists in a space or folder",
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

			var lists []clickup.List

			targetFolder := folderID
			if targetFolder == "" {
				cwd, _ := os.Getwd()
				targetFolder = cfg.FolderForDir(cwd)
			}

			if targetFolder != "" {
				// Get lists in a specific folder
				fetchedLists, _, err := client.Clickup.Lists.GetLists(context.Background(), targetFolder, false)
				if err != nil {
					return fmt.Errorf("failed to fetch lists for folder: %w", err)
				}
				lists = fetchedLists
			} else {
				// Get folderless lists in a space
				targetSpace := spaceID
				if targetSpace == "" {
					targetSpace = cfg.Space
				}

				if targetSpace == "" {
					return fmt.Errorf("no space or folder specified. Use --space, --folder, or run 'clickup space select'")
				}

				fetchedLists, _, err := client.Clickup.Lists.GetFolderlessLists(context.Background(), targetSpace, false)
				if err != nil {
					return fmt.Errorf("failed to fetch folderless lists for space: %w", err)
				}
				lists = fetchedLists
			}

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, lists)
			}

			if len(lists) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "No lists found.")
				return nil
			}

			tp := tableprinter.New(f.IOStreams)

			for _, list := range lists {
				tp.AddField(list.ID)
				tp.AddField(list.Name)
				tp.EndRow()
			}

			return tp.Render()
		},
	}

	cmd.Flags().StringVar(&spaceID, "space", "", "Space ID (defaults to configured space)")
	cmd.Flags().StringVar(&folderID, "folder", "", "Folder ID (lists inside a specific folder)")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)
	return cmd
}
