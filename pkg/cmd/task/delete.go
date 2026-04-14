package task

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type deleteOptions struct {
	taskIDs []string
	confirm bool
}

// NewCmdDelete returns a command to delete a ClickUp task.
func NewCmdDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <task-id> [<task-id>...]",
		Short: "Delete one or more tasks",
		Long: `Delete one or more ClickUp tasks permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.
Multiple task IDs can be provided for bulk deletion.`,
		Example: `  # Delete a task (with confirmation)
  clickup task delete 86a3xrwkp

  # Delete without confirmation
  clickup task delete CU-abc123 --yes

  # Bulk delete multiple tasks
  clickup task delete 86abc1 86abc2 86abc3 -y`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.taskIDs = args
			return runDelete(f, opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(f *cmdutil.Factory, opts *deleteOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	bulk := len(opts.taskIDs) > 1

	if !opts.confirm && ios.IsTerminal() {
		p := prompter.New(ios)
		var msg string
		if bulk {
			msg = fmt.Sprintf("Delete %d tasks?", len(opts.taskIDs))
		} else {
			parsed := git.ParseTaskID(opts.taskIDs[0])
			qs := cmdutil.CustomIDTaskQuery(cfg, parsed.IsCustomID)
			task, fetchErr := apiv2.GetTaskLocal(ctx, client, parsed.ID, qs)
			if fetchErr == nil {
				msg = fmt.Sprintf("Delete task %s (%s)?", cs.Bold(task.Name), parsed.ID)
			} else {
				msg = fmt.Sprintf("Delete task %s?", parsed.ID)
			}
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

	total := len(opts.taskIDs)
	var deleted int

	for i, rawID := range opts.taskIDs {
		parsed := git.ParseTaskID(rawID)
		taskID := parsed.ID
		qs := cmdutil.CustomIDTaskQuery(cfg, parsed.IsCustomID)

		// Fetch name for the output message (skip in bulk to halve API calls).
		name := taskID
		if !bulk {
			task, fetchErr := apiv2.GetTaskLocal(ctx, client, taskID, qs)
			if fetchErr == nil {
				name = task.Name
			}
		}

		err := apiv2.DeleteTaskLocal(ctx, client, taskID, qs)
		if err != nil {
			if bulk {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) failed to delete %s: %v\n", cs.Red("✗"), i+1, total, rawID, err)
				continue
			}
			return fmt.Errorf("failed to delete task %s: %w", taskID, err)
		}

		deleted++
		if bulk {
			fmt.Fprintf(ios.Out, "(%d/%d) Deleted task %s (%s)\n", i+1, total, cs.Bold(name), taskID)
		} else {
			fmt.Fprintf(ios.Out, "%s Task %s deleted (%s)\n", cs.Green("!"), cs.Bold(name), taskID)
		}
	}

	if bulk {
		fmt.Fprintf(ios.Out, "\n%s Deleted %d/%d tasks\n", cs.Green("!"), deleted, total)
	}

	return nil
}
