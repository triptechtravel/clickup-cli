package doc

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	clickupv3 "github.com/triptechtravel/clickup-cli/api/clickupv3"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type viewOptions struct {
	docID     string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdView returns a command to view a single ClickUp Doc.
func NewCmdView(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}

	cmd := &cobra.Command{
		Use:   "view <doc-id>",
		Short: "View a ClickUp Doc",
		Long:  `Display details about a ClickUp Doc including its metadata and parent location.`,
		Example: `  # View a Doc
  clickup doc view abc123

  # View as JSON
  clickup doc view abc123 --json`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.docID = args[0]
			return runView(f, opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runView(f *cmdutil.Factory, opts *viewOptions) error {
	ios := f.IOStreams

	workspaceID, err := resolveWorkspaceID(f)
	if err != nil {
		return err
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	d, err := apiv3.GetDocPublic(ctx, client, workspaceID, opts.docID)
	if err != nil {
		return fmt.Errorf("failed to fetch doc: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, d)
	}

	return printDocView(f, d)
}

func printDocView(f *cmdutil.Factory, d *clickupv3.PublicDocsDocDto) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	out := ios.Out

	fmt.Fprintf(out, "%s %s\n", cs.Bold(d.Name), cs.Gray("#"+d.ID))

	visibility := "private"
	if d.Public {
		visibility = "public"
	}
	fmt.Fprintf(out, "%s %s\n", cs.Bold("Visibility:"), visibility)

	if d.Parent.ID != "" {
		fmt.Fprintf(out, "%s %s (type %v)\n", cs.Bold("Parent:"), d.Parent.ID, d.Parent.Type)
	}

	deleted := d.Deleted != nil && *d.Deleted
	archived := d.Archived != nil && *d.Archived

	if deleted {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Status:"), cs.Red("deleted"))
	} else if archived {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Status:"), cs.Gray("archived"))
	}

	if d.DateCreated != 0 {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Created:"), text.FormatUnixMillisFloat(d.DateCreated))
	}
	if d.DateUpdated != nil && *d.DateUpdated != 0 {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Updated:"), text.FormatUnixMillisFloat(*d.DateUpdated))
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, cs.Gray("---"))
	fmt.Fprintln(out, cs.Gray("Quick actions:"))
	fmt.Fprintf(out, "  %s  clickup doc page list %s\n", cs.Gray("Pages:"), d.ID)
	fmt.Fprintf(out, "  %s  clickup doc page create %s --name \"My Page\"\n", cs.Gray("Add page:"), d.ID)
	fmt.Fprintf(out, "  %s  clickup doc view %s --json\n", cs.Gray("JSON:"), d.ID)

	return nil
}
