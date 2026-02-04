package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type editOptions struct {
	taskID              string
	name                string
	description         string
	markdownDescription string
	status              string
	priority            int
	assignees           []int
	removeAssignees     []int
	tags                []string
	dueDate             string
	startDate           string
	timeEstimate        string
	points              float64
	parent              string
	linksTo             string
	dueDateTime         bool
	startDateTime       bool
	notifyAll           bool
	customItemID        int
	fields              []string
	clearFields         []string
	jsonFlags           cmdutil.JSONFlags
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

Custom fields can be set with --field "Name=value" (repeatable) and cleared
with --clear-field "Name" (repeatable). Use 'clickup field list' to discover
available custom fields and their types.`,
		Example: `  # Update status and priority
  clickup task edit --status "in progress" --priority 2

  # Edit a specific task with a custom field
  clickup task edit CU-abc123 --field "Environment=production"

  # Set due date and time estimate
  clickup task edit --due-date 2025-03-01 --time-estimate 4h

  # Clear a custom field
  clickup task edit CU-abc123 --clear-field "Environment"`,
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
	cmd.Flags().StringVar(&opts.markdownDescription, "markdown-description", "", "New task description in markdown")
	cmd.Flags().StringVar(&opts.status, "status", "", "New task status")
	cmd.Flags().IntVar(&opts.priority, "priority", 0, "New task priority (1=Urgent, 2=High, 3=Normal, 4=Low)")
	cmd.Flags().IntSliceVar(&opts.assignees, "assignee", nil, "Assignee user ID(s) to add")
	cmd.Flags().IntSliceVar(&opts.removeAssignees, "remove-assignee", nil, "Assignee user ID(s) to remove")
	cmd.Flags().StringSliceVar(&opts.tags, "tags", nil, "Set tags (replaces existing)")
	cmd.Flags().StringVar(&opts.dueDate, "due-date", "", `Due date (YYYY-MM-DD, or "none" to clear)`)
	cmd.Flags().StringVar(&opts.startDate, "start-date", "", `Start date (YYYY-MM-DD, or "none" to clear)`)
	cmd.Flags().StringVar(&opts.timeEstimate, "time-estimate", "", `Time estimate (e.g. 2h, 30m, 1h30m; "0" to clear)`)
	cmd.Flags().Float64Var(&opts.points, "points", pointsNotSet, "Sprint/story points (-1 to clear)")
	cmd.Flags().StringVar(&opts.parent, "parent", "", "Parent task ID (make this a subtask)")
	cmd.Flags().StringVar(&opts.linksTo, "links-to", "", "Link to another task by ID")
	cmd.Flags().BoolVar(&opts.dueDateTime, "due-date-time", false, "Include time component in due date")
	cmd.Flags().BoolVar(&opts.startDateTime, "start-date-time", false, "Include time component in start date")
	cmd.Flags().BoolVar(&opts.notifyAll, "notify-all", false, "Notify all assignees and watchers")
	cmd.Flags().IntVar(&opts.customItemID, "type", -1, "Task type (0=task, 1=milestone, or custom type ID)")
	cmd.Flags().StringArrayVar(&opts.fields, "field", nil, `Set a custom field value ("Name=value", repeatable)`)
	cmd.Flags().StringArrayVar(&opts.clearFields, "clear-field", nil, `Clear a custom field value ("Name", repeatable)`)

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

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
	} else {
		parsed := git.ParseTaskID(taskID)
		taskID = parsed.ID
		isCustomID = parsed.IsCustomID
	}

	// Ensure at least one field is being updated.
	if !cmd.Flags().Changed("name") &&
		!cmd.Flags().Changed("description") &&
		!cmd.Flags().Changed("markdown-description") &&
		!cmd.Flags().Changed("status") &&
		!cmd.Flags().Changed("priority") &&
		!cmd.Flags().Changed("assignee") &&
		!cmd.Flags().Changed("remove-assignee") &&
		!cmd.Flags().Changed("tags") &&
		!cmd.Flags().Changed("due-date") &&
		!cmd.Flags().Changed("start-date") &&
		!cmd.Flags().Changed("time-estimate") &&
		!cmd.Flags().Changed("points") &&
		!cmd.Flags().Changed("parent") &&
		!cmd.Flags().Changed("links-to") &&
		!cmd.Flags().Changed("due-date-time") &&
		!cmd.Flags().Changed("start-date-time") &&
		!cmd.Flags().Changed("notify-all") &&
		!cmd.Flags().Changed("type") &&
		!cmd.Flags().Changed("field") &&
		!cmd.Flags().Changed("clear-field") {
		return fmt.Errorf("at least one field flag must be provided")
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	var getOpts *clickup.GetTaskOptions
	if isCustomID {
		getOpts = &clickup.GetTaskOptions{
			CustomTaskIDs: true,
		}
	}

	updateReq := &clickup.TaskUpdateRequest{}

	if cmd.Flags().Changed("name") {
		updateReq.Name = opts.name
	}
	if cmd.Flags().Changed("description") {
		updateReq.Description = opts.description
	}
	// markdown-description is handled via raw API after the main update call.
	if cmd.Flags().Changed("status") {
		// Validate status against the task's space statuses.
		fetchTask, _, fetchErr := client.Clickup.Tasks.GetTask(context.Background(), taskID, getOpts)
		if fetchErr == nil && fetchTask.Space.ID != "" {
			validated, valErr := cmdutil.ValidateStatus(client, fetchTask.Space.ID, opts.status, ios.ErrOut)
			if valErr != nil {
				return valErr
			}
			opts.status = validated
		}
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

	if cmd.Flags().Changed("parent") {
		updateReq.Parent = opts.parent
	}
	if cmd.Flags().Changed("links-to") {
		updateReq.LinksTo = opts.linksTo
	}
	if cmd.Flags().Changed("due-date-time") {
		updateReq.DueDateTime = opts.dueDateTime
	}
	if cmd.Flags().Changed("start-date-time") {
		updateReq.StartDateTime = opts.startDateTime
	}
	if cmd.Flags().Changed("notify-all") {
		updateReq.NotifyAll = opts.notifyAll
	}
	if cmd.Flags().Changed("type") {
		updateReq.CustomItemId = opts.customItemID
	}

	task, _, err := client.Clickup.Tasks.UpdateTask(context.Background(), taskID, getOpts, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update task %s: %w", taskID, err)
	}

	// Set points via raw API call if specified (not supported by go-clickup library).
	if cmd.Flags().Changed("points") {
		if err := setTaskPoints(client, task.ID, opts.points); err != nil {
			return fmt.Errorf("task updated but failed to set points: %w", err)
		}
	}

	// Set markdown description via raw API call (not supported by go-clickup library).
	if cmd.Flags().Changed("markdown-description") {
		if err := setMarkdownDescription(client, task.ID, opts.markdownDescription); err != nil {
			return fmt.Errorf("task updated but failed to set markdown description: %w", err)
		}
	}

	// Handle custom field set/clear operations.
	if cmd.Flags().Changed("field") {
		for _, fieldSpec := range opts.fields {
			parts := strings.SplitN(fieldSpec, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid --field format %q (use \"Name=value\")", fieldSpec)
			}
			fieldName, fieldValue := parts[0], parts[1]

			cf := resolveFieldByName(task.CustomFields, fieldName)
			if cf == nil {
				return fmt.Errorf("custom field %q not found (available: %s)", fieldName, customFieldNames(task.CustomFields))
			}

			parsed, err := parseFieldValue(cf, fieldValue)
			if err != nil {
				return err
			}

			_, err = client.Clickup.CustomFields.SetCustomFieldValue(context.Background(), task.ID, cf.ID, map[string]interface{}{"value": parsed}, nil)
			if err != nil {
				return fmt.Errorf("failed to set custom field %q: %w", fieldName, err)
			}
		}
	}

	if cmd.Flags().Changed("clear-field") {
		for _, fieldName := range opts.clearFields {
			cf := resolveFieldByName(task.CustomFields, fieldName)
			if cf == nil {
				return fmt.Errorf("custom field %q not found (available: %s)", fieldName, customFieldNames(task.CustomFields))
			}

			_, err = client.Clickup.CustomFields.RemoveCustomFieldValue(context.Background(), task.ID, cf.ID, nil)
			if err != nil {
				return fmt.Errorf("failed to clear custom field %q: %w", fieldName, err)
			}
		}
	}

	id := task.ID
	if task.CustomID != "" {
		id = task.CustomID
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, task)
	}

	fmt.Fprintf(ios.Out, "%s Updated task %s %s\n", cs.Green("!"), cs.Bold(task.Name), cs.Gray("#"+id))
	if task.URL != "" {
		fmt.Fprintf(ios.Out, "%s\n", cs.Cyan(task.URL))
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), id)
	fmt.Fprintf(ios.Out, "  %s  clickup status set <status> %s\n", cs.Gray("Status:"), id)
	fmt.Fprintf(ios.Out, "  %s  clickup comment add %s \"@user text\" (supports @mentions)\n", cs.Gray("Comment:"), id)

	return nil
}
