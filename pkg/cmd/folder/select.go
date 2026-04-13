package folder

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdFolderSelect returns the folder select command.
func NewCmdFolderSelect(f *cmdutil.Factory) *cobra.Command {
	var local bool

	cmd := &cobra.Command{
		Use:   "select",
		Short: "Select an active folder",
		Long: `Interactively select a folder to be your default.

When you select a folder, 'clickup list list' will automatically
show lists from this folder.

If --local is provided, the folder is set only for the current directory.
Otherwise, it sets the global default folder.`,
		Example: `  # Select a global default folder
  clickup folder select

  # Select a default folder only for the current directory
  clickup folder select --local`,
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

			targetSpace := cfg.Space
			if targetSpace == "" {
				return fmt.Errorf("no space configured. Run 'clickup space select' first")
			}

			folders, _, err := client.Clickup.Folders.GetFolders(context.Background(), targetSpace, false)
			if err != nil {
				return fmt.Errorf("failed to fetch folders: %w", err)
			}

			if len(folders) == 0 {
				return fmt.Errorf("no folders found in the configured space")
			}

			p := prompter.New(f.IOStreams)
			names := make([]string, len(folders))
			folderMap := make(map[string]string)

			for i, folder := range folders {
				displayName := fmt.Sprintf("%s (%s)", folder.Name, folder.ID)
				names[i] = displayName
				folderMap[displayName] = folder.ID
			}

			idx, err := p.Select("Select a folder", names)
			if err != nil {
				return err
			}

			selectedName := names[idx]
			selectedID := folderMap[selectedName]

			if local {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}

				dc := cfg.DirectoryDefaults[cwd]
				dc.Folder = selectedID
				cfg.SetDirectoryDefault(cwd, dc)

				if err := cfg.Save(); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				fmt.Fprintf(f.IOStreams.Out, "Saved local default folder: %s\n", selectedName)
			} else {
				cfg.Folder = selectedID
				if err := cfg.Save(); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				fmt.Fprintf(f.IOStreams.Out, "Saved global default folder: %s\n", selectedName)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Set folder only for the current directory")

	return cmd
}
