package chat

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type listOptions struct {
	jsonFlags cmdutil.JSONFlags
}

// NewCmdList returns the "chat list" command.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List ClickUp Chat channels",
		Long:  "List Chat channels in the configured workspace.",
		Example: `  # List all channels
  clickup chat list

  # List channels as JSON
  clickup chat list --json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(f, opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runList(f *cmdutil.Factory, opts *listOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	cfg, err := f.Config()
	if err != nil {
		return err
	}
	if cfg.Workspace == "" {
		return fmt.Errorf("no workspace configured; run 'clickup auth' first")
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	result, err := apiv3.ListChatChannels(context.Background(), client, cfg.Workspace)
	if err != nil {
		return fmt.Errorf("failed to list channels: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, result)
	}

	if len(result.Data) == 0 {
		fmt.Fprintln(ios.Out, cs.Gray("No channels found."))
		return nil
	}

	tp := tableprinter.New(ios)
	for _, ch := range result.Data {
		tp.AddField(ch.ID)
		tp.AddField(ch.Name)
		tp.AddField(string(ch.Type))
		tp.EndRow()
	}
	return tp.Render()
}
