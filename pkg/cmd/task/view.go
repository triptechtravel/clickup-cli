package task

import (
	"context"
	"fmt"
	"strings"
	"time"

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

	// Fetch the markdown description (the standard GetTask doesn't include it).
	var mdTask clickup.Task
	mdReq, err := client.Clickup.NewRequest("GET", fmt.Sprintf("task/%s/?include_markdown_description=true", task.ID), nil)
	if err == nil {
		if _, err := client.Clickup.Do(ctx, mdReq, &mdTask); err == nil && mdTask.MarkdownDescription != "" {
			task.MarkdownDescription = mdTask.MarkdownDescription
		}
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

	// Title with type badge
	id := task.ID
	if task.CustomID != "" {
		id = task.CustomID
	}
	typeBadge := ""
	if task.CustomItemId == 1 {
		typeBadge = cs.Yellow(" [Milestone]")
	} else if task.CustomItemId > 1 {
		typeBadge = cs.Gray(fmt.Sprintf(" [Type:%d]", task.CustomItemId))
	}
	fmt.Fprintf(out, "%s %s%s\n", cs.Bold(task.Name), cs.Gray("#"+id), typeBadge)

	// Location: Space > Folder > List
	if task.Space.ID != "" || task.Folder.Name != "" || task.List.Name != "" {
		var parts []string
		if task.Space.ID != "" {
			parts = append(parts, fmt.Sprintf("Space:%s", task.Space.ID))
		}
		if task.Folder.Name != "" && !task.Folder.Hidden {
			parts = append(parts, task.Folder.Name)
		}
		if task.List.Name != "" {
			parts = append(parts, task.List.Name)
		}
		if len(parts) > 0 {
			fmt.Fprintf(out, "%s %s\n", cs.Bold("Location:"), strings.Join(parts, " > "))
		}
	}

	// Parent task
	if task.Parent != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Parent:"), task.Parent)
	}

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

	// Watchers
	if len(task.Watchers) > 0 {
		names := make([]string, 0, len(task.Watchers))
		for _, w := range task.Watchers {
			names = append(names, w.Username)
		}
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Watchers:"), strings.Join(names, ", "))
	}

	// Tags
	if len(task.Tags) > 0 {
		tagNames := make([]string, 0, len(task.Tags))
		for _, t := range task.Tags {
			tagNames = append(tagNames, t.Name)
		}
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Tags:"), strings.Join(tagNames, ", "))
	}

	// Points
	if pts := task.Points.Value.String(); pts != "" && pts != "0" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Points:"), pts)
	}

	// Time Estimate & Time Spent
	if s := formatMillisDuration(task.TimeEstimate); s != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Time Estimate:"), s)
	}
	if s := formatMillisDuration(task.TimeSpent); s != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("Time Spent:"), s)
	}

	// Dates
	if task.DateCreated != "" {
		if t, err := parseUnixMillis(task.DateCreated); err == nil {
			fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Created:"), t.Format("2006-01-02 15:04"), text.RelativeTime(t))
		}
	}
	if task.DateUpdated != "" {
		if t, err := parseUnixMillis(task.DateUpdated); err == nil {
			fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Updated:"), t.Format("2006-01-02 15:04"), text.RelativeTime(t))
		}
	}
	if task.StartDate != "" {
		if t, err := parseUnixMillis(task.StartDate); err == nil {
			fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Start Date:"), t.Format("2006-01-02 15:04"), text.RelativeTime(t))
		}
	}
	if task.DueDate != nil {
		if dt := task.DueDate.Time(); dt != nil {
			fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Due:"), dt.Format("2006-01-02 15:04"), text.RelativeTime(*dt))
		}
	}
	if task.DateClosed != "" {
		if t, err := parseUnixMillis(task.DateClosed); err == nil {
			fmt.Fprintf(out, "%s %s (%s)\n", cs.Bold("Closed:"), t.Format("2006-01-02 15:04"), text.RelativeTime(t))
		}
	}

	// URL
	if task.URL != "" {
		fmt.Fprintf(out, "%s %s\n", cs.Bold("URL:"), cs.Cyan(task.URL))
	}

	// Dependencies
	if len(task.Dependencies) > 0 {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Dependencies:"))
		for _, dep := range task.Dependencies {
			if dep.DependsOn != "" {
				fmt.Fprintf(out, "  Waiting on: %s\n", dep.DependsOn)
			} else if dep.TaskID != "" {
				fmt.Fprintf(out, "  Blocking: %s\n", dep.TaskID)
			}
		}
	}

	// Linked Tasks
	if len(task.LinkedTasks) > 0 {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Linked Tasks:"))
		for _, lt := range task.LinkedTasks {
			fmt.Fprintf(out, "  %s\n", lt.TaskID)
		}
	}

	// Checklists
	if len(task.Checklists) > 0 {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Checklists:"))
		for _, cl := range task.Checklists {
			total := cl.Resolved + cl.Unresolved
			fmt.Fprintf(out, "  %s (%d/%d)\n", cs.Bold(cl.Name), cl.Resolved, total)
			for _, item := range cl.Items {
				marker := "[ ]"
				if item.Resolved {
					marker = "[x]"
				}
				fmt.Fprintf(out, "    %s %s\n", marker, item.Name)
			}
		}
	}

	// Attachments
	if len(task.Attachments) > 0 {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Attachments:"))
		for _, att := range task.Attachments {
			if att.Title != "" {
				fmt.Fprintf(out, "  %s", att.Title)
			} else {
				fmt.Fprintf(out, "  (attachment)")
			}
			if att.Url != "" {
				fmt.Fprintf(out, " %s", cs.Cyan(att.Url))
			}
			fmt.Fprintln(out)
		}
	}

	// Custom Fields
	if len(task.CustomFields) > 0 {
		var hasValues bool
		for _, cf := range task.CustomFields {
			if v := formatCustomFieldValue(cf); v != "" {
				hasValues = true
				break
			}
		}
		if hasValues {
			fmt.Fprintf(out, "\n%s\n", cs.Bold("Custom Fields:"))
			for _, cf := range task.CustomFields {
				if v := formatCustomFieldValue(cf); v != "" {
					fmt.Fprintf(out, "  %s: %s\n", cf.Name, v)
				}
			}
		}
	}

	// Description (prefer markdown for richer content including URLs)
	desc := task.MarkdownDescription
	if desc == "" {
		desc = task.Description
	}
	if desc != "" {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Description:"))
		fmt.Fprintf(out, "%s\n", text.IndentLines(desc, "  "))
	}

	// Quick actions footer
	fmt.Fprintln(out)
	fmt.Fprintln(out, cs.Gray("---"))
	fmt.Fprintln(out, cs.Gray("Quick actions:"))
	fmt.Fprintf(out, "  %s  clickup task edit %s --status <status>\n", cs.Gray("Edit:"), id)
	fmt.Fprintf(out, "  %s  clickup status set <status> %s\n", cs.Gray("Status:"), id)
	fmt.Fprintf(out, "  %s  clickup comment add %s \"text\"\n", cs.Gray("Comment:"), id)
	fmt.Fprintf(out, "  %s  clickup link pr --task %s\n", cs.Gray("Link PR:"), id)
	fmt.Fprintf(out, "  %s  clickup task view %s --json\n", cs.Gray("JSON:"), id)

	return nil
}

// formatMillisDuration converts milliseconds to a human-readable duration string.
func formatMillisDuration(ms int64) string {
	if ms <= 0 {
		return ""
	}
	d := time.Duration(ms) * time.Millisecond
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	if m > 0 {
		return fmt.Sprintf("%dm", m)
	}
	return "< 1m"
}
