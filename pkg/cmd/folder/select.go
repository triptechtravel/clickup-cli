package folder

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/config"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdFolderSelect returns the folder select command.
func NewCmdFolderSelect(f *cmdutil.Factory) *cobra.Command {
	var (
		local   bool
		spaceID string
	)

	cmd := &cobra.Command{
		Use:   "select",
		Short: "Set default folder",
		Long:  "Set the default folder globally or for the current directory.",
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

			folders, err := apiv2.GetFoldersLocal(context.Background(), client, spaceID, false)
			if err != nil {
				return fmt.Errorf("failed to fetch folders: %w", err)
			}

			if len(folders) == 0 {
				return fmt.Errorf("no folders found in space %s", spaceID)
			}

			p := prompter.New(f.IOStreams)
			names := make([]string, len(folders))
			for i, folder := range folders {
				names[i] = fmt.Sprintf("%s (%s)", folder.Name, folder.ID)
			}

			idx, err := p.Select("Select a folder:", names)
			if err != nil {
				return err
			}

			selectedID := folders[idx].ID
			selectedName := folders[idx].Name

			if local {
				dir, err := os.Getwd()
				if err != nil {
					return err
				}
				dc := config.DirectoryConfig{Folder: selectedID}
				// Preserve existing directory defaults.
				if cfg.DirectoryDefaults != nil {
					if existing, ok := cfg.DirectoryDefaults[dir]; ok {
						dc.Space = existing.Space
						dc.List = existing.List
					}
				}
				cfg.SetDirectoryDefault(dir, dc)
				if err := cfg.Save(); err != nil {
					return err
				}
				fmt.Fprintf(f.IOStreams.Out, "Default folder for %s set to %s (%s)\n", dir, selectedName, selectedID)
			} else {
				cfg.Folder = selectedID
				if err := cfg.Save(); err != nil {
					return err
				}
				fmt.Fprintf(f.IOStreams.Out, "Default folder set to %s (%s)\n", selectedName, selectedID)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Set folder only for the current directory")
	cmd.Flags().StringVar(&spaceID, "space", "", "Space ID (defaults to configured space)")

	return cmd
}
