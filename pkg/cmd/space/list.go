package space

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdSpaceList returns the space list command.
func NewCmdSpaceList(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List spaces in your workspace",
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

			if jsonFlags.WantsJSON() {
				return jsonFlags.OutputJSON(f.IOStreams.Out, spaces)
			}

			if len(spaces) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "No spaces found.")
				return nil
			}

			cs := f.IOStreams.ColorScheme()
			tp := tableprinter.New(f.IOStreams)

			for _, s := range spaces {
				active := ""
				if s.ID == cfg.Space {
					active = cs.Green("●")
				}
				tp.AddField(active)
				tp.AddField(s.ID)
				tp.AddField(s.Name)
				tp.EndRow()
			}

			if err := tp.Render(); err != nil {
				return err
			}

			// Quick actions footer
			fmt.Fprintln(f.IOStreams.Out)
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("---"))
			fmt.Fprintln(f.IOStreams.Out, cs.Gray("Quick actions:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup space select\n", cs.Gray("Select:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup task recent\n", cs.Gray("Recent:"))
			fmt.Fprintf(f.IOStreams.Out, "  %s  clickup space list --json\n", cs.Gray("JSON:"))

			return nil
		},
	}

	cmdutil.AddJSONFlags(cmd, &jsonFlags)
	return cmd
}
