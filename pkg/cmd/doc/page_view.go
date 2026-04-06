package doc

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type pageViewOptions struct {
	docID         string
	pageID        string
	contentFormat string
	jsonFlags     cmdutil.JSONFlags
}

// NewCmdPageView returns a command to view a single page in a ClickUp Doc.
func NewCmdPageView(f *cmdutil.Factory) *cobra.Command {
	opts := &pageViewOptions{}

	cmd := &cobra.Command{
		Use:   "view <doc-id> <page-id>",
		Short: "View a page in a ClickUp Doc",
		Long:  `Display the content and metadata of a specific page within a ClickUp Doc.`,
		Example: `  # View a page
  clickup doc page view abc123 page456

  # View as markdown
  clickup doc page view abc123 page456 --content-format text/md

  # View as JSON
  clickup doc page view abc123 page456 --json`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.docID = args[0]
			opts.pageID = args[1]
			if opts.contentFormat != "" && !containsString(validContentFormats, opts.contentFormat) {
				return fmt.Errorf("invalid content format %q; valid values: %s", opts.contentFormat, strings.Join(validContentFormats, "|"))
			}
			return runPageView(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.contentFormat, "content-format", "", "Content format for page body (text/md|text/plain)")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runPageView(f *cmdutil.Factory, opts *pageViewOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	workspaceID, err := resolveWorkspaceID(f)
	if err != nil {
		return err
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	var params []apiv3.GetPagePublicParams
	if opts.contentFormat != "" {
		params = append(params, apiv3.GetPagePublicParams{ContentFormat: opts.contentFormat})
	}
	p, err := apiv3.GetPagePublic(ctx, client, workspaceID, opts.docID, opts.pageID, params...)
	if err != nil {
		return fmt.Errorf("failed to fetch page: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, p)
	}

	out := ios.Out
	fmt.Fprintf(out, "%s %s\n", cs.Bold(p.Name), cs.Gray("#"+p.Id))
	if p.SubTitle != nil && *p.SubTitle != "" {
		fmt.Fprintf(out, "%s\n", cs.Gray(*p.SubTitle))
	}
	if p.DateCreated != 0 {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Created:"), text.FormatUnixMillisFloat(p.DateCreated))
	}
	if p.DateUpdated != nil && *p.DateUpdated != 0 {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Updated:"), text.FormatUnixMillisFloat(*p.DateUpdated))
	}

	if p.Pages != nil && len(*p.Pages) > 0 {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Sub-pages:"))
		printPageTree(out, *p.Pages, 1, cs)
	}

	if p.Content != "" {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Content:"))
		fmt.Fprintln(out, p.Content)
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, cs.Gray("---"))
	fmt.Fprintln(out, cs.Gray("Quick actions:"))
	fmt.Fprintf(out, "  %s  clickup doc page edit %s %s --content \"...\"\n", cs.Gray("Edit:"), opts.docID, p.Id)
	fmt.Fprintf(out, "  %s  clickup doc page view %s %s --json\n", cs.Gray("JSON:"), opts.docID, p.Id)

	return nil
}
