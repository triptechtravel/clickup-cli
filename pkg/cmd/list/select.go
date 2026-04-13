package list

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdListSelect returns the list select command.
func NewCmdListSelect(f *cmdutil.Factory) *cobra.Command {
	var local bool
	var folderID string

	cmd := &cobra.Command{
		Use:   "select",
		Short: "Select an active list",
		Long: `Interactively select a list to be your default.

The selected list will be used by default for task commands.

If --local is provided, the list is set only for the current directory.
Otherwise, it sets the global default list.`,
		Example: `  # Select a global default list
  clickup list select

  # Select a list from a specific folder
  clickup list select --folder 12345

  # Select a default list only for the current directory
  clickup list select --local`,
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
			
			targetFolder := folderID
			if targetFolder == "" {
				cwd, _ := os.Getwd()
				targetFolder = cfg.FolderForDir(cwd)
			}
			
			if targetSpace == "" && targetFolder == "" {
				return fmt.Errorf("no space or folder configured. Run 'clickup space select' or 'clickup folder select' first")
			}

			// For simplicity in interactive selection, we fetch lists depending on folder vs space.
			var listsMap map[string]string = make(map[string]string)
			var listNames []string

			if targetFolder != "" {
				lists, _, err := client.Clickup.Lists.GetLists(context.Background(), targetFolder, false)
				if err != nil {
					return fmt.Errorf("failed to fetch lists for folder: %w", err)
				}
				for _, list := range lists {
					listsMap[list.Name] = list.ID
					listNames = append(listNames, list.Name)
				}
			} else {
				// To keep it simple, just fetching folderless lists for space.
				lists, _, err := client.Clickup.Lists.GetFolderlessLists(context.Background(), targetSpace, false)
				if err != nil {
					return fmt.Errorf("failed to fetch lists for space: %w", err)
				}
				for _, list := range lists {
					listsMap[list.Name] = list.ID
					listNames = append(listNames, list.Name)
				}
			}

			if len(listNames) == 0 {
				return fmt.Errorf("no lists found in the selected space/folder")
			}

			p := prompter.New(f.IOStreams)
			names := make([]string, len(listNames))
			for i, name := range listNames {
				names[i] = fmt.Sprintf("%s (%s)", name, listsMap[name])
			}

			idx, err := p.Select("Select a list", names)
			if err != nil {
				return err
			}

			selectedName := listNames[idx]
			selectedID := listsMap[selectedName]

			if local {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}

				dc := cfg.DirectoryDefaults[cwd]
				dc.List = selectedID
				cfg.SetDirectoryDefault(cwd, dc)

				if err := cfg.Save(); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				fmt.Fprintf(f.IOStreams.Out, "Saved local default list: %s (%s)\n", selectedName, selectedID)
			} else {
				cfg.List = selectedID
				if err := cfg.Save(); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				fmt.Fprintf(f.IOStreams.Out, "Saved global default list: %s (%s)\n", selectedName, selectedID)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Set list only for the current directory")
	cmd.Flags().StringVar(&folderID, "folder", "", "Folder ID to search lists in")

	return cmd
}
