package space

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
	teamID    string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdSpaceCreate returns a command to create a new space.
func NewCmdSpaceCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new space",
		Long: `Create a new ClickUp space in your workspace.

The --name flag is required. If --team is not provided, the configured
workspace is used.`,
		Example: `  # Create a space
  clickup space create --name "Dev"

  # Create in a specific workspace
  clickup space create --name "Dev" --team 12345`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSpaceCreate(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Space name (required)")
	cmd.Flags().StringVar(&opts.teamID, "team", "", "Workspace/team ID (defaults to configured workspace)")
	_ = cmd.MarkFlagRequired("name")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runSpaceCreate(f *cmdutil.Factory, opts *createOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	teamID := opts.teamID
	if teamID == "" {
		cfg, err := f.Config()
		if err != nil {
			return err
		}
		teamID = cfg.Workspace
		if teamID == "" {
			return fmt.Errorf("no workspace configured. Run 'clickup auth login' first or pass --team")
		}
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	req := &clickupv2.CreateSpaceJSONRequest{
		Name: opts.name,
	}

	resp, err := apiv2.CreateSpace(ctx, client, teamID, req)
	if err != nil {
		return fmt.Errorf("failed to create space: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, resp)
	}

	fmt.Fprintf(ios.Out, "%s Created space %s (%s)\n", cs.Green("!"), cs.Bold(resp.Name), resp.ID)

	return nil
}
