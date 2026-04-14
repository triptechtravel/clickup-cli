package list

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
	"github.com/triptechtravel/clickup-cli/internal/config"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdListSelect returns the list select command.
func NewCmdListSelect(f *cmdutil.Factory) *cobra.Command {
	var (
		local    bool
		folderID string
		spaceID  string
	)

	cmd := &cobra.Command{
		Use:   "select",
		Short: "Set default list",
		Long:  "Set the default list globally or for the current directory.",
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
				lists, err = apiv2.GetListsLocal(ctx, client, folderID, false)
				if err != nil {
					return fmt.Errorf("failed to fetch lists: %w", err)
				}
			} else if spaceID != "" {
				lists, err = apiv2.GetFolderlessListsLocal(ctx, client, spaceID, false)
				if err != nil {
					return fmt.Errorf("failed to fetch folderless lists: %w", err)
				}
			} else {
				return fmt.Errorf("no folder or space configured. Use --folder, --space, or run 'clickup folder select' / 'clickup space select' first")
			}

			if len(lists) == 0 {
				return fmt.Errorf("no lists found")
			}

			p := prompter.New(f.IOStreams)
			names := make([]string, len(lists))
			for i, l := range lists {
				names[i] = fmt.Sprintf("%s (%s)", l.Name, l.ID)
			}

			idx, err := p.Select("Select a list:", names)
			if err != nil {
				return err
			}

			selectedID := lists[idx].ID
			selectedName := lists[idx].Name

			if local {
				dir, err := os.Getwd()
				if err != nil {
					return err
				}
				dc := config.DirectoryConfig{List: selectedID}
				// Preserve existing directory defaults.
				if cfg.DirectoryDefaults != nil {
					if existing, ok := cfg.DirectoryDefaults[dir]; ok {
						dc.Space = existing.Space
						dc.Folder = existing.Folder
					}
				}
				cfg.SetDirectoryDefault(dir, dc)
				if err := cfg.Save(); err != nil {
					return err
				}
				fmt.Fprintf(f.IOStreams.Out, "Default list for %s set to %s (%s)\n", dir, selectedName, selectedID)
			} else {
				cfg.List = selectedID
				if err := cfg.Save(); err != nil {
					return err
				}
				fmt.Fprintf(f.IOStreams.Out, "Default list set to %s (%s)\n", selectedName, selectedID)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Set list only for the current directory")
	cmd.Flags().StringVar(&folderID, "folder", "", "Folder ID (defaults to configured folder)")
	cmd.Flags().StringVar(&spaceID, "space", "", "Space ID (defaults to configured space, used for folderless lists)")

	return cmd
}
