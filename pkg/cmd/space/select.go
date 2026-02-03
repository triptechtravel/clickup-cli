package space

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/config"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdSpaceSelect returns the space select command.
func NewCmdSpaceSelect(f *cmdutil.Factory) *cobra.Command {
	var directory bool

	cmd := &cobra.Command{
		Use:   "select [NAME]",
		Short: "Set default space",
		Long:  "Set the default space globally or for the current directory.",
		Args:  cobra.MaximumNArgs(1),
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

			teamID := cfg.Workspace
			if teamID == "" {
				return fmt.Errorf("no workspace configured. Run 'clickup auth login' first")
			}

			spaces, _, err := client.Clickup.Spaces.GetSpaces(context.Background(), teamID, false)
			if err != nil {
				return fmt.Errorf("failed to fetch spaces: %w", err)
			}

			if len(spaces) == 0 {
				return fmt.Errorf("no spaces found in workspace")
			}

			var selectedID, selectedName string

			if len(args) > 0 {
				// Match by name
				search := args[0]
				for _, s := range spaces {
					if s.Name == search || s.ID == search {
						selectedID = s.ID
						selectedName = s.Name
						break
					}
				}
				if selectedID == "" {
					return fmt.Errorf("space %q not found. Use 'clickup space list' to see available spaces", search)
				}
			} else {
				// Interactive selection
				p := prompter.New(f.IOStreams)
				names := make([]string, len(spaces))
				for i, s := range spaces {
					names[i] = fmt.Sprintf("%s (%s)", s.Name, s.ID)
				}

				idx, err := p.Select("Select a space:", names)
				if err != nil {
					return err
				}
				selectedID = spaces[idx].ID
				selectedName = spaces[idx].Name
			}

			if directory {
				dir, err := os.Getwd()
				if err != nil {
					return err
				}
				cfg.SetDirectoryDefault(dir, config.DirectoryConfig{Space: selectedID})
				if err := cfg.Save(); err != nil {
					return err
				}
				fmt.Fprintf(f.IOStreams.Out, "Default space for %s set to %s (%s)\n", dir, selectedName, selectedID)
			} else {
				cfg.Space = selectedID
				if err := cfg.Save(); err != nil {
					return err
				}
				fmt.Fprintf(f.IOStreams.Out, "Default space set to %s (%s)\n", selectedName, selectedID)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&directory, "directory", false, "Set space only for the current directory")
	return cmd
}
