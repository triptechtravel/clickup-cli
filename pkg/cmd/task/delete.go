package task

import (
	"context"
	"fmt"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type deleteOptions struct {
	taskID  string
	confirm bool
}

// NewCmdDelete returns a command to delete a ClickUp task.
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <task-id>",
		Short: "Delete a task",
		Long: `Delete a ClickUp task permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.`,
		Example: `  # Delete a task (with confirmation)
  clickup task delete 86a3xrwkp

  # Delete without confirmation
  clickup task delete CU-abc123 --yes`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.taskID = args[0]
			return runDelete(f, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(f *cmdutil.Factory, opts *deleteOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	parsed := git.ParseTaskID(opts.taskID)
	taskID := parsed.ID

	var getOpts *clickup.GetTaskOptions
	if parsed.IsCustomID {
		getOpts = &clickup.GetTaskOptions{
			CustomTaskIDs: true,
		}
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	// Fetch task name for the confirmation prompt.
	ctx := context.Background()
	task, _, fetchErr := client.Clickup.Tasks.GetTask(ctx, taskID, getOpts)

	if !opts.confirm && ios.IsTerminal() {
		p := prompter.New(ios)
		msg := fmt.Sprintf("Delete task %s?", taskID)
		if fetchErr == nil {
			msg = fmt.Sprintf("Delete task %s (%s)?", cs.Bold(task.Name), taskID)
		}
		ok, err := p.Confirm(msg, false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(ios.ErrOut, "Cancelled.")
			return nil
		}
	}

	_, err = client.Clickup.Tasks.DeleteTask(ctx, taskID, getOpts)
	if err != nil {
		return fmt.Errorf("failed to delete task %s: %w", taskID, err)
	}

	name := taskID
	if fetchErr == nil {
		name = task.Name
	}

	fmt.Fprintf(ios.Out, "%s Task %s deleted (%s)\n", cs.Green("!"), cs.Bold(name), taskID)
	return nil
}
