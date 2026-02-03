package task

import (
	"context"
	"fmt"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type editOptions struct {
	taskID      string
	name        string
	description string
	status      string
	priority    int
	assignees   []int
}

// NewCmdEdit returns a command to edit an existing ClickUp task.
func NewCmdEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &editOptions{}

	cmd := &cobra.Command{
		Use:   "edit [<task-id>]",
		Short: "Edit a ClickUp task",
		Long: `Edit an existing ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. At least one field flag must be provided.`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.taskID = args[0]
			}
			return runEdit(f, opts, cmd)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "New task name")
	cmd.Flags().StringVar(&opts.description, "description", "", "New task description")
	cmd.Flags().StringVar(&opts.status, "status", "", "New task status")
	cmd.Flags().IntVar(&opts.priority, "priority", 0, "New task priority (1=Urgent, 2=High, 3=Normal, 4=Low)")
	cmd.Flags().IntSliceVar(&opts.assignees, "assignee", nil, "Assignee user ID(s) to set")

	return cmd
}

func runEdit(f *cmdutil.Factory, opts *editOptions, cmd *cobra.Command) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	taskID := opts.taskID
	isCustomID := false

	// Auto-detect task ID from git branch if not provided.
	if taskID == "" {
		gitCtx, err := f.GitContext()
		if err != nil {
			return fmt.Errorf("could not detect task ID: %w\n\n%s", err, git.BranchNamingSuggestion(""))
		}
		if gitCtx.TaskID == nil {
			fmt.Fprintln(ios.ErrOut, cs.Yellow(git.BranchNamingSuggestion(gitCtx.Branch)))
			return &cmdutil.SilentError{Err: fmt.Errorf("no task ID found in branch")}
		}
		taskID = gitCtx.TaskID.ID
		isCustomID = gitCtx.TaskID.IsCustomID
	}

	// Ensure at least one field is being updated.
	if !cmd.Flags().Changed("name") &&
		!cmd.Flags().Changed("description") &&
		!cmd.Flags().Changed("status") &&
		!cmd.Flags().Changed("priority") &&
		!cmd.Flags().Changed("assignee") {
		return fmt.Errorf("at least one field flag must be provided (--name, --description, --status, --priority, --assignee)")
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	updateReq := &clickup.TaskUpdateRequest{}

	if cmd.Flags().Changed("name") {
		updateReq.Name = opts.name
	}
	if cmd.Flags().Changed("description") {
		updateReq.Description = opts.description
	}
	if cmd.Flags().Changed("status") {
		updateReq.Status = opts.status
	}
	if cmd.Flags().Changed("priority") {
		updateReq.Priority = opts.priority
	}
	if cmd.Flags().Changed("assignee") {
		updateReq.Assignees = clickup.TaskAssigneeUpdateRequest{
			Add: opts.assignees,
		}
	}

	var getOpts *clickup.GetTaskOptions
	if isCustomID {
		getOpts = &clickup.GetTaskOptions{
			CustomTaskIDs: true,
		}
	}

	ctx := context.Background()
	task, _, err := client.Clickup.Tasks.UpdateTask(ctx, taskID, getOpts, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update task %s: %w", taskID, err)
	}

	id := task.ID
	if task.CustomID != "" {
		id = task.CustomID
	}

	fmt.Fprintf(ios.Out, "%s Updated task %s %s\n", cs.Green("!"), cs.Bold(task.Name), cs.Gray("#"+id))
	if task.URL != "" {
		fmt.Fprintf(ios.Out, "%s\n", cs.Cyan(task.URL))
	}

	return nil
}
