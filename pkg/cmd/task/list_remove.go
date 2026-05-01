package task

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type listRemoveOptions struct {
	taskIDs   []string
	listID    string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdListRemove returns a command to remove tasks from an additional list.
func NewCmdListRemove(f *cmdutil.Factory) *cobra.Command {
	opts := &listRemoveOptions{}

	cmd := &cobra.Command{
		Use:   "list-remove <task-id>... --list-id <list-id>",
		Short: "Remove tasks from a list",
		Long: `Remove one or more tasks from a ClickUp list.

This removes the task's membership in the specified list. The task must
belong to at least one other list — you cannot remove a task from its
only list.

Multiple task IDs can be provided to remove them all from the same list.
Each task is processed independently; errors on individual tasks are
reported but do not stop the batch.`,
		Example: `  # Remove a single task from a sprint list
  clickup task list-remove 86abc123 --list-id 901613544162

  # Remove multiple tasks from the same list
  clickup task list-remove 86abc1 86abc2 86abc3 --list-id 901613544162`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.taskIDs = cmdutil.ExpandIDArgs(args)
			if len(opts.taskIDs) == 0 {
				return fmt.Errorf("no task IDs provided")
			}
			return runListRemove(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.listID, "list-id", "", "List ID to remove from (required)")
	_ = cmd.MarkFlagRequired("list-id")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runListRemove(f *cmdutil.Factory, opts *listRemoveOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	type result struct {
		TaskID string `json:"task_id"`
		ListID string `json:"list_id"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	bulk := len(opts.taskIDs) > 1
	total := len(opts.taskIDs)
	var removed int
	var results []result

	for i, rawID := range opts.taskIDs {
		parsed := git.ParseTaskID(rawID)
		taskID := parsed.ID

		err := removeTaskFromList(client, opts.listID, taskID)
		if err != nil {
			results = append(results, result{TaskID: rawID, ListID: opts.listID, Status: "error", Error: err.Error()})
			if bulk {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: %v\n", cs.Yellow("!"), i+1, total, rawID, err)
				continue
			}
			return fmt.Errorf("failed to remove task %s from list %s: %w", rawID, opts.listID, err)
		}

		removed++
		results = append(results, result{TaskID: rawID, ListID: opts.listID, Status: "removed"})

		if !opts.jsonFlags.WantsJSON() {
			if bulk {
				fmt.Fprintf(ios.Out, "(%d/%d) Removed task %s from list %s\n", i+1, total, cs.Bold(rawID), cs.Cyan(opts.listID))
			} else {
				fmt.Fprintf(ios.Out, "%s Removed task %s from list %s\n", cs.Green("!"), cs.Bold(rawID), cs.Cyan(opts.listID))
			}
		}
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, results)
	}

	if bulk {
		fmt.Fprintf(ios.Out, "\n%s Removed %d/%d tasks from list %s\n", cs.Green("!"), removed, total, cs.Cyan(opts.listID))
	}

	return nil
}
