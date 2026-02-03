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
	taskID          string
	name            string
	description     string
	status          string
	priority        int
	assignees       []int
	removeAssignees []int
	tags            []string
	dueDate         string
	startDate       string
	timeEstimate    string
	points          float64
}

// NewCmdEdit returns a command to edit an existing ClickUp task.
func NewCmdEdit(f *cmdutil.Factory) *cobra.Command {
	opts := &editOptions{}

	cmd := &cobra.Command{
		Use:   "edit [<task-id>]",
		Short: "Edit a ClickUp task",
		Long: `Edit an existing ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. At least one field flag must be provided.

Supported fields:
  --name              New task name
  --description       New task description
  --status            New task status
  --priority          New priority (1=Urgent, 2=High, 3=Normal, 4=Low)
  --assignee          Assignee user ID(s) to add
  --remove-assignee   Assignee user ID(s) to remove
  --tags              Set tags (replaces existing)
  --due-date          Due date in YYYY-MM-DD format (use "none" to clear)
  --start-date        Start date in YYYY-MM-DD format (use "none" to clear)
  --time-estimate     Time estimate (e.g. "2h", "30m", "1h30m"; use "0" to clear)
  --points            Sprint/story points (use -1 to clear)`,
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
	cmd.Flags().IntSliceVar(&opts.assignees, "assignee", nil, "Assignee user ID(s) to add")
	cmd.Flags().IntSliceVar(&opts.removeAssignees, "remove-assignee", nil, "Assignee user ID(s) to remove")
	cmd.Flags().StringSliceVar(&opts.tags, "tags", nil, "Set tags (replaces existing)")
	cmd.Flags().StringVar(&opts.dueDate, "due-date", "", `Due date (YYYY-MM-DD, or "none" to clear)`)
	cmd.Flags().StringVar(&opts.startDate, "start-date", "", `Start date (YYYY-MM-DD, or "none" to clear)`)
	cmd.Flags().StringVar(&opts.timeEstimate, "time-estimate", "", `Time estimate (e.g. 2h, 30m, 1h30m; "0" to clear)`)
	cmd.Flags().Float64Var(&opts.points, "points", pointsNotSet, "Sprint/story points (-1 to clear)")

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
		!cmd.Flags().Changed("assignee") &&
		!cmd.Flags().Changed("remove-assignee") &&
		!cmd.Flags().Changed("tags") &&
		!cmd.Flags().Changed("due-date") &&
		!cmd.Flags().Changed("start-date") &&
		!cmd.Flags().Changed("time-estimate") &&
		!cmd.Flags().Changed("points") {
		return fmt.Errorf("at least one field flag must be provided (--name, --description, --status, --priority, --assignee, --remove-assignee, --tags, --due-date, --start-date, --time-estimate, --points)")
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
	if cmd.Flags().Changed("assignee") || cmd.Flags().Changed("remove-assignee") {
		updateReq.Assignees = clickup.TaskAssigneeUpdateRequest{
			Add: opts.assignees,
			Rem: opts.removeAssignees,
		}
	}

	if cmd.Flags().Changed("tags") {
		updateReq.Tags = opts.tags
	}

	if cmd.Flags().Changed("due-date") {
		if opts.dueDate == "none" {
			updateReq.DueDate = clickup.NullDate()
		} else {
			d, err := parseDate(opts.dueDate)
			if err != nil {
				return err
			}
			updateReq.DueDate = d
		}
	}

	if cmd.Flags().Changed("start-date") {
		if opts.startDate == "none" {
			updateReq.StartDate = clickup.NullDate()
		} else {
			d, err := parseDate(opts.startDate)
			if err != nil {
				return err
			}
			updateReq.StartDate = d
		}
	}

	if cmd.Flags().Changed("time-estimate") {
		if opts.timeEstimate == "0" {
			updateReq.TimeEstimate = 0
		} else {
			ms, err := parseDuration(opts.timeEstimate)
			if err != nil {
				return err
			}
			updateReq.TimeEstimate = ms
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

	// Set points via raw API call if specified (not supported by go-clickup library).
	if cmd.Flags().Changed("points") {
		pointsTaskID := task.ID
		if err := setTaskPoints(client, pointsTaskID, opts.points); err != nil {
			return fmt.Errorf("task updated but failed to set points: %w", err)
		}
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
