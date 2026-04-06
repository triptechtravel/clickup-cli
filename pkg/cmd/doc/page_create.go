package doc

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	clickupv3 "github.com/triptechtravel/clickup-cli/api/clickupv3"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type pageCreateOptions struct {
	docID         string
	name          string
	parentPageID  string
	subTitle      string
	content       string
	contentFormat string
	jsonFlags     cmdutil.JSONFlags
}

// NewCmdPageCreate returns a command to create a page within a ClickUp Doc.
func NewCmdPageCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &pageCreateOptions{}

	cmd := &cobra.Command{
		Use:   "create <doc-id>",
		Short: "Create a page in a ClickUp Doc",
		Long: `Create a new page within a ClickUp Doc.

Pages can be nested under other pages using --parent-page-id.
Content can be provided as plain text or markdown.`,
		Example: `  # Create a basic page
  clickup doc page create abc123 --name "Introduction"

  # Create a page with markdown content
  clickup doc page create abc123 --name "Setup Guide" \
    --content "# Setup\n\nFollow these steps..." --content-format text/md

  # Create a nested page
  clickup doc page create abc123 --name "Advanced Config" \
    --parent-page-id page456

  # Create and output JSON
  clickup doc page create abc123 --name "Release Notes" --json`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.docID = args[0]
			if opts.name == "" {
				return fmt.Errorf("--name is required")
			}
			if opts.contentFormat != "" && !containsString(validContentFormats, opts.contentFormat) {
				return fmt.Errorf("invalid content format %q; valid values: %s", opts.contentFormat, strings.Join(validContentFormats, "|"))
			}
			return runPageCreate(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Page name (required)")
	cmd.Flags().StringVar(&opts.parentPageID, "parent-page-id", "", "Parent page ID (for nested pages)")
	cmd.Flags().StringVar(&opts.subTitle, "sub-title", "", "Page subtitle")
	cmd.Flags().StringVar(&opts.content, "content", "", "Page content")
	cmd.Flags().StringVar(&opts.contentFormat, "content-format", "", "Content format (text/md|text/plain)")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runPageCreate(f *cmdutil.Factory, opts *pageCreateOptions) error {
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

	req := &clickupv3.PublicDocsPublicCreatePageOptionsDto{
		Name: &opts.name,
	}
	if opts.parentPageID != "" {
		req.ParentPageID = &opts.parentPageID
	}
	if opts.subTitle != "" {
		req.SubTitle = &opts.subTitle
	}
	if opts.content != "" {
		req.Content = &opts.content
	}
	if opts.contentFormat != "" {
		req.ContentFormat = &opts.contentFormat
	}

	ctx := context.Background()
	p, err := apiv3.CreatePagePublic(ctx, client, workspaceID, opts.docID, req)
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, p)
	}

	fmt.Fprintf(ios.Out, "%s Created page %s %s\n", cs.Green("!"), cs.Bold(p.Name), cs.Gray("#"+p.ID))

	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup doc page view %s %s\n", cs.Gray("View:"), opts.docID, p.ID)
	fmt.Fprintf(ios.Out, "  %s  clickup doc page edit %s %s --content \"...\"\n", cs.Gray("Edit:"), opts.docID, p.ID)

	return nil
}
