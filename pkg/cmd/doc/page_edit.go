package doc

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type pageEditOptions struct {
	docID           string
	pageID          string
	name            string
	subTitle        string
	content         string
	contentFormat   string
	contentEditMode string
	jsonFlags       cmdutil.JSONFlags
}

// NewCmdPageEdit returns a command to edit an existing page in a ClickUp Doc.
func NewCmdPageEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &pageEditOptions{}

	cmd := &cobra.Command{
		Use:   "edit <doc-id> <page-id>",
		Short: "Edit a page in a ClickUp Doc",
		Long: `Update the name, subtitle, or content of a page in a ClickUp Doc.

Use --content-edit-mode to control whether content replaces, appends to, or
prepends to the existing page content.`,
		Example: `  # Replace page content
  clickup doc page edit abc123 page456 --content "# New content"

  # Append to existing content
  clickup doc page edit abc123 page456 \
    --content "## Release Notes\n\n- Fixed bug X" \
    --content-edit-mode append

  # Rename a page
  clickup doc page edit abc123 page456 --name "Updated Title"

  # Edit and output JSON
  clickup doc page edit abc123 page456 --content "Updated" --json`,
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.docID = args[0]
			opts.pageID = args[1]
			if opts.contentFormat != "" && !containsString(validContentFormats, opts.contentFormat) {
				return fmt.Errorf("invalid content format %q; valid values: %s", opts.contentFormat, strings.Join(validContentFormats, "|"))
			}
			if opts.contentEditMode != "" && !containsString(validEditModes, strings.ToLower(opts.contentEditMode)) {
				return fmt.Errorf("invalid content edit mode %q; valid values: %s", opts.contentEditMode, strings.Join(validEditModes, "|"))
			}
			if opts.name == "" && opts.subTitle == "" && opts.content == "" {
				return fmt.Errorf("at least one of --name, --sub-title, or --content is required")
			}
			return runPageEdit(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "New page name")
	cmd.Flags().StringVar(&opts.subTitle, "sub-title", "", "New page subtitle")
	cmd.Flags().StringVar(&opts.content, "content", "", "Page content")
	cmd.Flags().StringVar(&opts.contentFormat, "content-format", "", "Content format (text/md|text/plain)")
	cmd.Flags().StringVar(&opts.contentEditMode, "content-edit-mode", "replace", "How to apply content (replace|append|prepend)")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runPageEdit(f *cmdutil.Factory, opts *pageEditOptions) error {
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

	req := &apiv3.EditPageRequest{}
	if opts.name != "" {
		req.Name = opts.name
	}
	if opts.subTitle != "" {
		req.SubTitle = opts.subTitle
	}
	if opts.content != "" {
		req.Content = opts.content
		if opts.contentFormat != "" {
			req.ContentFormat = opts.contentFormat
		}
		if opts.contentEditMode != "" {
			req.ContentEditMode = strings.ToLower(opts.contentEditMode)
		}
	}

	ctx := context.Background()
	p, err := apiv3.EditPage(ctx, client, workspaceID, opts.docID, opts.pageID, req)
	if err != nil {
		return fmt.Errorf("failed to edit page: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, p)
	}

	name := p.Name
	if name == "" {
		name = opts.pageID
	}
	fmt.Fprintf(ios.Out, "%s Updated page %s %s\n", cs.Green("!"), cs.Bold(name), cs.Gray("#"+opts.pageID))

	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup doc page view %s %s\n", cs.Gray("View:"), opts.docID, opts.pageID)
	fmt.Fprintf(ios.Out, "  %s  clickup doc page view %s %s --json\n", cs.Gray("JSON:"), opts.docID, opts.pageID)

	return nil
}
