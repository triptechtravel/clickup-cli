package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type editOptions struct {
	taskIDs             []string
	name                string
	description         string
	markdownDescription string
	status              string
	priority            int
	assignees           []int
	removeAssignees     []int
	tags                []string
	addTags             []string
	removeTags          []string
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
		Use:   "edit [<task-id>...]",
		Short: "Edit a ClickUp task",
		Long: `Edit one or more existing ClickUp tasks.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. At least one field flag must be provided.

Multiple task IDs can be provided to apply the same changes to all tasks.
Each task is updated independently; errors on individual tasks are reported
but do not stop the batch.

Custom fields can be set with --field "Name=value" (repeatable) and cleared
with --clear-field "Name" (repeatable). Use 'clickup field list' to discover
available custom fields and their types.`,
		Example: `  # Update status and priority (auto-detects task from git branch)
  clickup task edit --status "in progress" --priority 2

  # Edit a specific task
  clickup task edit CU-abc123 --field "Environment=production"
  clickup task edit CU-abc123 --due-date 2025-03-01 --time-estimate 4h
  clickup task edit CU-abc123 --clear-field "Environment"

  # Bulk edit: close multiple subtasks at once
  clickup task edit 86abc1 86abc2 86abc3 --status "Closed"

  # Bulk edit: set due date on many tasks
  clickup task edit 86abc1 86abc2 86abc3 --due-date 2026-03-01

  # Add tags without removing existing ones
  clickup task edit CU-abc123 --add-tags new-feature-development
  clickup task edit 86abc1 86abc2 --add-tags r&d,new-app-development

  # Remove specific tags
  clickup task edit CU-abc123 --remove-tags fix`,
		Args:              cobra.ArbitraryArgs,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.taskIDs = args
			return runEdit(f, opts, cmd)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "New task name (convention: [Type] Context — Action (Platform))")
	cmd.Flags().StringVar(&opts.description, "description", "", "New task description")
	cmd.Flags().StringVar(&opts.markdownDescription, "markdown-description", "", "New task description in markdown")
	cmd.Flags().StringVar(&opts.status, "status", "", "New task status")
	cmd.Flags().IntVar(&opts.priority, "priority", 0, "New task priority (1=Urgent, 2=High, 3=Normal, 4=Low)")
	cmd.Flags().IntSliceVar(&opts.assignees, "assignee", nil, "Assignee user ID(s) to add")
	cmd.Flags().IntSliceVar(&opts.removeAssignees, "remove-assignee", nil, "Assignee user ID(s) to remove")
	cmd.Flags().StringSliceVar(&opts.tags, "tags", nil, "Set tags (replaces existing)")
	cmd.Flags().StringSliceVar(&opts.addTags, "add-tags", nil, "Add tags without removing existing ones")
	cmd.Flags().StringSliceVar(&opts.removeTags, "remove-tags", nil, "Remove specific tags")
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

	// Resolve task IDs: auto-detect from git branch if none provided.
	taskIDs := opts.taskIDs
	if len(taskIDs) == 0 {
		gitCtx, err := f.GitContext()
		if err != nil {
			return fmt.Errorf("could not detect task ID: %w\n\n%s", err, git.BranchNamingSuggestion(""))
		}
		if gitCtx.TaskID == nil {
			fmt.Fprintln(ios.ErrOut, cs.Yellow(git.BranchNamingSuggestion(gitCtx.Branch)))
			return &cmdutil.SilentError{Err: fmt.Errorf("no task ID found in branch")}
		}
		taskIDs = []string{gitCtx.TaskID.ID}
		if gitCtx.TaskID.IsCustomID {
			// Mark that this single auto-detected ID is custom.
			taskIDs[0] = gitCtx.TaskID.ID
		}
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
		!cmd.Flags().Changed("add-tags") &&
		!cmd.Flags().Changed("remove-tags") &&
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

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	// Build the update request once (shared across all tasks).
	updateReq := &clickup.TaskUpdateRequest{}

	if cmd.Flags().Changed("name") {
		updateReq.Name = opts.name
	}
	if cmd.Flags().Changed("description") {
		updateReq.Description = opts.description
	}

	// Validate status and tags against the first task's space/list (shared across batch).
	var spaceID, listID string
	if cmd.Flags().Changed("status") || cmd.Flags().Changed("tags") || cmd.Flags().Changed("add-tags") {
		parsed := git.ParseTaskID(taskIDs[0])
		qs := cmdutil.CustomIDTaskQuery(cfg, parsed.IsCustomID)
		fetchTask, fetchErr := apiv2.GetTaskLocal(context.Background(), client, parsed.ID, qs)
		if fetchErr == nil && fetchTask.Space.ID != "" {
			spaceID = fetchTask.Space.ID
			listID = fetchTask.List.ID
		}
	}

	if cmd.Flags().Changed("status") {
		if spaceID != "" {
			validated, valErr := cmdutil.ValidateStatusWithList(client, spaceID, listID, opts.status, ios.ErrOut)
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
		if spaceID != "" {
			opts.tags = cmdutil.ValidateTags(client, spaceID, opts.tags, ios.ErrOut)
		}
	}
	if cmd.Flags().Changed("add-tags") {
		if spaceID != "" {
			opts.addTags = cmdutil.EnsureTagsExist(client, spaceID, opts.addTags, ios.ErrOut)
		}
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

	// Validate --field format before the loop.
	if cmd.Flags().Changed("field") {
		for _, fieldSpec := range opts.fields {
			parts := strings.SplitN(fieldSpec, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid --field format %q (use \"Name=value\")", fieldSpec)
			}
		}
	}

	bulk := len(taskIDs) > 1
	total := len(taskIDs)
	var updated int
	var results []*clickup.Task

	for i, rawID := range taskIDs {
		parsed := git.ParseTaskID(rawID)
		taskID := parsed.ID
		qs := cmdutil.CustomIDTaskQuery(cfg, parsed.IsCustomID)

		task, err := apiv2.UpdateTaskLocal(context.Background(), client, taskID, updateReq, qs)
		if err != nil {
			if bulk {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: %v\n", cs.Yellow("!"), i+1, total, rawID, err)
				continue
			}
			return fmt.Errorf("failed to update task %s: %w", rawID, err)
		}

		// Set tags via dedicated API calls (replaces existing tags with desired set).
		if cmd.Flags().Changed("tags") {
			var currentTagNames []string
			for _, tag := range task.Tags {
				currentTagNames = append(currentTagNames, tag.Name)
			}
			if err := setTaskTags(client, task.ID, currentTagNames, opts.tags); err != nil {
				if bulk {
					fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: updated but failed to set tags: %v\n", cs.Yellow("!"), i+1, total, rawID, err)
				} else {
					return fmt.Errorf("task updated but failed to set tags: %w", err)
				}
			}
		}

		// Add tags incrementally (without removing existing ones).
		if cmd.Flags().Changed("add-tags") {
			for _, t := range opts.addTags {
				if err := addTaskTag(client, task.ID, t); err != nil {
					if bulk {
						fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: failed to add tag %q: %v\n", cs.Yellow("!"), i+1, total, rawID, t, err)
					} else {
						return fmt.Errorf("failed to add tag %q: %w", t, err)
					}
				}
			}
		}

		// Remove specific tags.
		if cmd.Flags().Changed("remove-tags") {
			for _, t := range opts.removeTags {
				if err := removeTaskTag(client, task.ID, t); err != nil {
					if bulk {
						fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: failed to remove tag %q: %v\n", cs.Yellow("!"), i+1, total, rawID, t, err)
					} else {
						return fmt.Errorf("failed to remove tag %q: %w", t, err)
					}
				}
			}
		}

		// Set points via raw API call.
		if cmd.Flags().Changed("points") {
			if err := setTaskPoints(client, task.ID, opts.points); err != nil {
				if bulk {
					fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: updated but failed to set points: %v\n", cs.Yellow("!"), i+1, total, rawID, err)
				} else {
					return fmt.Errorf("task updated but failed to set points: %w", err)
				}
			}
		}

		// Set markdown description via raw API call.
		if cmd.Flags().Changed("markdown-description") {
			if err := setMarkdownDescription(client, task.ID, opts.markdownDescription); err != nil {
				if bulk {
					fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: updated but failed to set markdown description: %v\n", cs.Yellow("!"), i+1, total, rawID, err)
				} else {
					return fmt.Errorf("task updated but failed to set markdown description: %w", err)
				}
			}
		}

		// Handle custom field set/clear operations.
		if cmd.Flags().Changed("field") {
			for _, fieldSpec := range opts.fields {
				parts := strings.SplitN(fieldSpec, "=", 2)
				fieldName, fieldValue := parts[0], parts[1]

				cf := resolveFieldByName(task.CustomFields, fieldName)
				if cf == nil {
					if bulk {
						fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: custom field %q not found\n", cs.Yellow("!"), i+1, total, rawID, fieldName)
					} else {
						return fmt.Errorf("custom field %q not found (available: %s)", fieldName, customFieldNames(task.CustomFields))
					}
					continue
				}

				parsed, err := parseFieldValue(cf, fieldValue, newUserResolver(context.Background(), client))
				if err != nil {
					if bulk {
						fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: custom field %q: %v\n", cs.Yellow("!"), i+1, total, rawID, fieldName, err)
					} else {
						return err
					}
					continue
				}

				err = apiv2.SetCustomFieldValueLocal(context.Background(), client, task.ID, cf.ID, parsed, "")
				if err != nil {
					if bulk {
						fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: failed to set custom field %q: %v\n", cs.Yellow("!"), i+1, total, rawID, fieldName, err)
					} else {
						return fmt.Errorf("failed to set custom field %q: %w", fieldName, err)
					}
				}
			}
		}

		if cmd.Flags().Changed("clear-field") {
			for _, fieldName := range opts.clearFields {
				cf := resolveFieldByName(task.CustomFields, fieldName)
				if cf == nil {
					if bulk {
						fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: custom field %q not found\n", cs.Yellow("!"), i+1, total, rawID, fieldName)
					} else {
						return fmt.Errorf("custom field %q not found (available: %s)", fieldName, customFieldNames(task.CustomFields))
					}
					continue
				}

				err = apiv2.RemoveCustomFieldValueLocal(context.Background(), client, task.ID, cf.ID, "")
				if err != nil {
					if bulk {
						fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: failed to clear custom field %q: %v\n", cs.Yellow("!"), i+1, total, rawID, fieldName, err)
					} else {
						return fmt.Errorf("failed to clear custom field %q: %w", fieldName, err)
					}
				}
			}
		}

		id := task.ID
		if task.CustomID != "" {
			id = task.CustomID
		}

		updated++
		results = append(results, task)

		if !opts.jsonFlags.WantsJSON() {
			if bulk {
				fmt.Fprintf(ios.Out, "(%d/%d) Updated task %s %s\n", i+1, total, cs.Bold(task.Name), cs.Gray("#"+id))
			} else {
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
			}
		}
	}

	if opts.jsonFlags.WantsJSON() {
		if bulk {
			return opts.jsonFlags.OutputJSON(ios.Out, results)
		}
		if len(results) > 0 {
			return opts.jsonFlags.OutputJSON(ios.Out, results[0])
		}
	}

	if bulk {
		fmt.Fprintf(ios.Out, "\n%s Updated %d/%d tasks\n", cs.Green("!"), updated, total)
	}

	return nil
}
