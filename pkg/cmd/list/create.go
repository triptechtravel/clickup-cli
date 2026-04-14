package list

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	clickupv2 "github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type createOptions struct {
	name      string
	folderID  string
	spaceID   string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdListCreate returns a command to create a new list.
func NewCmdListCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new list",
		Long: `Create a new ClickUp list in a folder or space.

Use --folder to create a list inside a folder.
Use --space to create a folderless list directly in a space.
One of --folder or --space is required.`,
		Example: `  # Create a list in a folder
  clickup list create --name "Backlog" --folder 12345

  # Create a folderless list in a space
  clickup list create --name "Backlog" --space 67890`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.folderID == "" && opts.spaceID == "" {
				// Fall back to configured space.
				cfg, cfgErr := f.Config()
				if cfgErr != nil {
					return cfgErr
				}
				if cfg.Space != "" {
					opts.spaceID = cfg.Space
				} else {
					return fmt.Errorf("either --folder or --space is required")
				}
			}
			return runListCreate(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "List name (required)")
	cmd.Flags().StringVar(&opts.folderID, "folder", "", "Folder ID to create the list in")
	cmd.Flags().StringVar(&opts.spaceID, "space", "", "Space ID to create a folderless list in")
	_ = cmd.MarkFlagRequired("name")

	cmd.MarkFlagsMutuallyExclusive("folder", "space")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runListCreate(f *cmdutil.Factory, opts *createOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if opts.folderID != "" {
		req := &clickupv2.CreateListJSONRequest{
			Name: opts.name,
		}

		resp, err := apiv2.CreateList(ctx, client, opts.folderID, req)
		if err != nil {
			return fmt.Errorf("failed to create list: %w", err)
		}

		if opts.jsonFlags.WantsJSON() {
			return opts.jsonFlags.OutputJSON(ios.Out, resp)
		}

		id := ""
		name := opts.name
		if resp.ID != nil {
			id = *resp.ID
		}
		if resp.Name != nil {
			name = *resp.Name
		}

		fmt.Fprintf(ios.Out, "%s Created list %s (%s)\n", cs.Green("!"), cs.Bold(name), id)
		return nil
	}

	// Folderless list in a space.
	req := &clickupv2.CreateFolderlessListJSONRequest{
		Name: opts.name,
	}

	resp, err := apiv2.CreateFolderlessList(ctx, client, opts.spaceID, req)
	if err != nil {
		return fmt.Errorf("failed to create list: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, resp)
	}

	id := ""
	name := opts.name
	if resp.ID != nil {
		id = *resp.ID
	}
	if resp.Name != nil {
		name = *resp.Name
	}

	fmt.Fprintf(ios.Out, "%s Created list %s (%s)\n", cs.Green("!"), cs.Bold(name), id)
	return nil
}
