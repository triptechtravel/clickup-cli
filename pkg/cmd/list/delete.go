package list

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type deleteOptions struct {
	listID  string
	confirm bool
}

// NewCmdListDelete returns a command to delete a list.
func NewCmdListDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <list-id>",
		Short: "Delete a list",
		Long: `Delete a ClickUp list permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.`,
		Example: `  # Delete a list (with confirmation)
  clickup list delete 12345

  # Delete without confirmation
  clickup list delete 12345 --yes`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.listID = args[0]
			return runListDelete(f, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runListDelete(f *cmdutil.Factory, opts *deleteOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if !opts.confirm && ios.IsTerminal() {
		name := opts.listID
		list, fetchErr := apiv2.GetList(ctx, client, opts.listID)
		if fetchErr == nil && list.Name != nil {
			name = *list.Name
		}

		p := prompter.New(ios)
		msg := fmt.Sprintf("Delete list %s (%s)?", cs.Bold(name), opts.listID)
		ok, err := p.Confirm(msg, false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(ios.ErrOut, "Cancelled.")
			return nil
		}
	}

	_, err = apiv2.DeleteList(ctx, client, opts.listID)
	if err != nil {
		return fmt.Errorf("failed to delete list %s: %w", opts.listID, err)
	}

	fmt.Fprintf(ios.Out, "%s List %s deleted\n", cs.Green("!"), opts.listID)

	return nil
}
