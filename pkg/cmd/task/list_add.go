package task

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type listAddOptions struct {
	taskIDs   []string
	listID    string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdListAdd returns a command to add tasks to an additional list.
func NewCmdListAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &listAddOptions{}

	cmd := &cobra.Command{
		Use:   "list-add <task-id>... --list-id <list-id>",
		Short: "Add tasks to an additional list",
		Long: `Add one or more tasks to an additional ClickUp list.

This does not move the task — it adds the task to the specified list
as a secondary location, so the task appears in multiple lists.

Multiple task IDs can be provided to add them all to the same list.
Each task is processed independently; errors on individual tasks are
reported but do not stop the batch.`,
		Example: `  # Add a single task to a sprint list
  clickup task list-add 86abc123 --list-id 901613544162

  # Add multiple tasks to the same list
  clickup task list-add 86abc1 86abc2 86abc3 --list-id 901613544162`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.taskIDs = cmdutil.ExpandIDArgs(args)
			if len(opts.taskIDs) == 0 {
				return fmt.Errorf("no task IDs provided")
			}
			return runListAdd(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.listID, "list-id", "", "Target list ID (required)")
	_ = cmd.MarkFlagRequired("list-id")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runListAdd(f *cmdutil.Factory, opts *listAddOptions) error {
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
	var added int
	var results []result

	for i, rawID := range opts.taskIDs {
		parsed := git.ParseTaskID(rawID)
		taskID := parsed.ID

		err := addTaskToList(client, opts.listID, taskID)
		if err != nil {
			results = append(results, result{TaskID: rawID, ListID: opts.listID, Status: "error", Error: err.Error()})
			if bulk {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: %v\n", cs.Yellow("!"), i+1, total, rawID, err)
				continue
			}
			return fmt.Errorf("failed to add task %s to list %s: %w", rawID, opts.listID, err)
		}

		added++
		results = append(results, result{TaskID: rawID, ListID: opts.listID, Status: "added"})

		if !opts.jsonFlags.WantsJSON() {
			if bulk {
				fmt.Fprintf(ios.Out, "(%d/%d) Added task %s to list %s\n", i+1, total, cs.Bold(rawID), cs.Cyan(opts.listID))
			} else {
				fmt.Fprintf(ios.Out, "%s Added task %s to list %s\n", cs.Green("!"), cs.Bold(rawID), cs.Cyan(opts.listID))
			}
		}
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, results)
	}

	if bulk {
		fmt.Fprintf(ios.Out, "\n%s Added %d/%d tasks to list %s\n", cs.Green("!"), added, total, cs.Cyan(opts.listID))
	}

	return nil
}
