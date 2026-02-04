package tag

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type listOptions struct {
	spaceID   string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdTagList returns the "tag list" command.
func NewCmdTagList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tags in a space",
		Long: `Display all tags available in a ClickUp space.

Uses the default space from your config unless --space-id is provided.`,
		Example: `  # List tags for the default space
  clickup tag list

  # List tags for a specific space
  clickup tag list --space-id 12345678

  # Output as JSON
  clickup tag list --json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTagList(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.spaceID, "space-id", "", "Space ID (defaults to configured space)")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runTagList(f *cmdutil.Factory, opts *listOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	spaceID := opts.spaceID
	if spaceID == "" {
		cfg, err := f.Config()
		if err != nil {
			return err
		}
		cwd, _ := os.Getwd()
		spaceID = cfg.SpaceForDir(cwd)
		if spaceID == "" {
			return fmt.Errorf("no space configured. Use --space-id or run 'clickup config set space <id>'")
		}
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	tags, err := cmdutil.FetchSpaceTags(client, spaceID)
	if err != nil {
		return err
	}

	if opts.jsonFlags.WantsJSON() {
		type tagJSON struct {
			Name string `json:"name"`
		}
		data := make([]tagJSON, len(tags))
		for i, t := range tags {
			data[i] = tagJSON{Name: t}
		}
		return opts.jsonFlags.OutputJSON(ios.Out, data)
	}

	if len(tags) == 0 {
		fmt.Fprintf(ios.Out, "No tags found for space %s\n", spaceID)
		return nil
	}

	tp := tableprinter.New(ios)

	tp.AddField(cs.Bold("NAME"))
	tp.EndRow()

	for _, t := range tags {
		tp.AddField(t)
		tp.EndRow()
	}

	if err := tp.Render(); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "\n%s\n", cs.Gray(fmt.Sprintf("Total: %d tags", len(tags))))

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task edit <id> --tags tag1,tag2\n", cs.Gray("Set:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task create --list-id <id> --tags tag1,tag2\n", cs.Gray("Create:"))
	fmt.Fprintf(ios.Out, "  %s  clickup tag list --json\n", cs.Gray("JSON:"))

	return nil
}
