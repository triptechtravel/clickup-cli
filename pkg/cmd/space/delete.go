package space

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type deleteOptions struct {
	spaceID string
	confirm bool
}

// NewCmdSpaceDelete returns a command to delete a space.
func NewCmdSpaceDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <space-id>",
		Short: "Delete a space",
		Long: `Delete a ClickUp space permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.`,
		Example: `  # Delete a space (with confirmation)
  clickup space delete 12345

  # Delete without confirmation
  clickup space delete 12345 --yes`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.spaceID = args[0]
			return runSpaceDelete(f, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runSpaceDelete(f *cmdutil.Factory, opts *deleteOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if !opts.confirm && ios.IsTerminal() {
		// Try to fetch name for display.
		name := opts.spaceID
		space, fetchErr := apiv2.GetSpace(ctx, client, opts.spaceID)
		if fetchErr == nil {
			name = space.Name
		}

		p := prompter.New(ios)
		msg := fmt.Sprintf("Delete space %s (%s)?", cs.Bold(name), opts.spaceID)
		ok, err := p.Confirm(msg, false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(ios.ErrOut, "Cancelled.")
			return nil
		}
	}

	_, err = apiv2.DeleteSpace(ctx, client, opts.spaceID)
	if err != nil {
		return fmt.Errorf("failed to delete space %s: %w", opts.spaceID, err)
	}

	fmt.Fprintf(ios.Out, "%s Space %s deleted\n", cs.Green("!"), opts.spaceID)

	return nil
}
