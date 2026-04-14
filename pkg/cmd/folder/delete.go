package folder

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type deleteOptions struct {
	folderID string
	confirm  bool
}

// NewCmdFolderDelete returns a command to delete a folder.
func NewCmdFolderDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <folder-id>",
		Short: "Delete a folder",
		Long: `Delete a ClickUp folder permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.`,
		Example: `  # Delete a folder (with confirmation)
  clickup folder delete 12345

  # Delete without confirmation
  clickup folder delete 12345 --yes`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.folderID = args[0]
			return runFolderDelete(f, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runFolderDelete(f *cmdutil.Factory, opts *deleteOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if !opts.confirm && ios.IsTerminal() {
		name := opts.folderID
		folder, fetchErr := apiv2.GetFolder(ctx, client, opts.folderID)
		if fetchErr == nil {
			name = folder.Name
		}

		p := prompter.New(ios)
		msg := fmt.Sprintf("Delete folder %s (%s)?", cs.Bold(name), opts.folderID)
		ok, err := p.Confirm(msg, false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(ios.ErrOut, "Cancelled.")
			return nil
		}
	}

	_, err = apiv2.DeleteFolder(ctx, client, opts.folderID)
	if err != nil {
		return fmt.Errorf("failed to delete folder %s: %w", opts.folderID, err)
	}

	fmt.Fprintf(ios.Out, "%s Folder %s deleted\n", cs.Green("!"), opts.folderID)

	return nil
}
