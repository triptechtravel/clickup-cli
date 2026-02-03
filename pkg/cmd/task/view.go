package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type viewOptions struct {
	taskID    string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdView returns a command to view a single ClickUp task.
func NewCmdView(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}

	cmd := &cobra.Command{
		Use:   "view [<task-id>]",
		Short: "View a ClickUp task",
		Long: `Display detailed information about a ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. Branch names containing CU-<hex> or
PREFIX-<number> patterns are recognized.`,
		Example: `  # View a specific task
  clickup task view 86a3xrwkp

  # Auto-detect task from git branch
  clickup task view

  # Output as JSON
  clickup task view 86a3xrwkp --json`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.taskID = args[0]
			}
			return runView(f, opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runView(f *cmdutil.Factory, opts *viewOptions) error {
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

	ctx := context.Background()
	task, _, err := client.Clickup.Tasks.GetTask(ctx, taskID, getOpts)
	if err != nil {
		return fmt.Errorf("failed to fetch task %s: %w", taskID, err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, task)
	}

	return printTaskView(f, task)
}

func printTaskView(f *cmdutil.Factory, task *clickup.Task) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	out := ios.Out

	// Title
	id := task.ID
	if task.CustomID != "" {
		id = task.CustomID
	}
	fmt.Fprintf(out, "%s %s\n", cs.Bold(task.Name), cs.Gray("#"+id))

	// Status
	statusText := task.Status.Status
	statusColorFn := cs.StatusColor(strings.ToLower(statusText))
	fmt.Fprintf(out, "%s %s\n", cs.Bold("Status:"), statusColorFn(statusText))

	// Priority
	if task.Priority.Priority != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Priority:"), task.Priority.Priority)
	}

	// Assignees
	if len(task.Assignees) > 0 {
		names := make([]string, 0, len(task.Assignees))
		for _, a := range task.Assignees {
			names = append(names, a.Username)
		}
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Assignees:"), strings.Join(names, ", "))
	}

	// Tags
	if len(task.Tags) > 0 {
		tagNames := make([]string, 0, len(task.Tags))
		for _, t := range task.Tags {
			tagNames = append(tagNames, t.Name)
		}
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Tags:"), strings.Join(tagNames, ", "))
	}

	// Dates
	if task.DateCreated != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Created:"), task.DateCreated)
	}
	if task.DateUpdated != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Updated:"), task.DateUpdated)
	}
	if task.DueDate != nil {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Due:"), task.DueDate.String())
	}

	// URL
	if task.URL != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("URL:"), cs.Cyan(task.URL))
	}

	// Description
	if task.Description != "" {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Description:"))
		fmt.Fprintf(out, "%s\n", text.IndentLines(task.Description, "  "))
	}

	return nil
}
