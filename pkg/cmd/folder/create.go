package folder

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	clickupv2 "github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type createOptions struct {
	name    string
	spaceID string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdFolderCreate returns a command to create a new folder.
func NewCmdFolderCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new folder",
		Long: `Create a new ClickUp folder in a space.

The --name flag is required. If --space is not provided, the configured
space is used.`,
		Example: `  # Create a folder in the current space
  clickup folder create --name "Sprint Folder"

  # Create in a specific space
  clickup folder create --name "Sprint Folder" --space 67890`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFolderCreate(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Folder name (required)")
	cmd.Flags().StringVar(&opts.spaceID, "space", "", "Space ID (defaults to configured space)")
	_ = cmd.MarkFlagRequired("name")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runFolderCreate(f *cmdutil.Factory, opts *createOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	spaceID := opts.spaceID
	if spaceID == "" {
		cfg, err := f.Config()
		if err != nil {
			return err
		}
		spaceID = cfg.Space
		if spaceID == "" {
			return fmt.Errorf("no space configured. Run 'clickup space select' first or pass --space")
		}
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	req := &clickupv2.CreateFolderJSONRequest{
		Name: opts.name,
	}

	resp, err := apiv2.CreateFolder(ctx, client, spaceID, req)
	if err != nil {
		return fmt.Errorf("failed to create folder: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, resp)
	}

	fmt.Fprintf(ios.Out, "%s Created folder %s (%s)\n", cs.Green("!"), cs.Bold(resp.Name), resp.ID)

	return nil
}
