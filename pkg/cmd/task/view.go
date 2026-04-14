package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/iostreams"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type subtaskInfo struct {
	ID       string `json:"id"`
	CustomID string `json:"custom_id"`
	Name     string `json:"name"`
	Status   struct {
		Status string `json:"status"`
	} `json:"status"`
	Assignees []struct {
		Username string `json:"username"`
	} `json:"assignees"`
	StartDate string        `json:"start_date"`
	DueDate   *clickup.Date `json:"due_date"`
	Subtasks  []subtaskInfo `json:"subtasks,omitempty"`
}

type taskWithExtras struct {
	clickup.Task
	Subtasks []subtaskInfo `json:"subtasks"`
}

type viewOptions struct {
	taskIDs   []string
	recursive bool
	jsonFlags cmdutil.JSONFlags
}

// NewCmdView returns a command to view one or more ClickUp tasks.
func NewCmdView(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}

	cmd := &cobra.Command{
		Use:   "view [<task-id>...]",
		Short: "View one or more ClickUp tasks",
		Long: `Display detailed information about one or more ClickUp tasks.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. Branch names containing CU-<id> or
PREFIX-<number> patterns are recognized.

If no task ID is found in the branch name, the command checks for an
associated GitHub PR and searches task descriptions for the PR URL.

Multiple task IDs can be provided for bulk fetching. In bulk mode, tasks are
fetched concurrently (up to 10 parallel requests). Bulk mode requires JSON
output (--json, --jq, or --template) and returns an array of tasks.`,
		Example: `  # View a specific task
  clickup task view 86a3xrwkp

  # Auto-detect task from git branch
  clickup task view

  # Output as JSON (includes subtasks with IDs, dates, and statuses)
  clickup task view 86a3xrwkp --json

  # View with recursive subtasks (fetches all descendants)
  clickup task view 86a3xrwkp --recursive --json

  # Bulk fetch multiple tasks as JSON array
  clickup task view 86abc1 86abc2 86abc3 --json

  # Extract tags from multiple tasks
  clickup task view 86abc1 86abc2 --jq '.[].tags[].name'

  # Extract subtask IDs for bulk operations
  clickup task view 86parent --json  # then use .subtasks[].id`,
		Args:              cobra.ArbitraryArgs,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.taskIDs = args
			if len(args) > 1 && !opts.jsonFlags.WantsJSON() {
				return fmt.Errorf("bulk view requires JSON output: add --json, --jq, or --template")
			}
			return runView(f, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.recursive, "recursive", false, "Recursively fetch all descendant subtasks")
	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runView(f *cmdutil.Factory, opts *viewOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	taskIDs := opts.taskIDs

	// Auto-detect task ID from git branch if none provided.
	if len(taskIDs) == 0 {
		gitCtx, err := f.GitContext()
		if err != nil {
			return fmt.Errorf("could not detect task ID: %w\n\n%s", err, git.BranchNamingSuggestion(""))
		}
		if gitCtx.TaskID == nil {
			// Try to find task via current branch's PR URL in task descriptions.
			if foundID, _, prNum := findTaskViaPR(f, ios); foundID != "" {
				fmt.Fprintf(ios.ErrOut, "Detected task %s via PR #%d\n", cs.Bold(foundID), prNum)
				taskIDs = []string{foundID}
			} else {
				fmt.Fprintln(ios.ErrOut, cs.Yellow(git.BranchNamingSuggestion(gitCtx.Branch)))
				return &cmdutil.SilentError{Err: fmt.Errorf("no task ID found in branch")}
			}
		} else {
			taskIDs = []string{gitCtx.TaskID.ID}
		}
	}

	// Bulk mode: fetch multiple tasks concurrently.
	if len(taskIDs) > 1 {
		return runViewBulk(f, opts, taskIDs)
	}

	// Single task mode (original behavior).
	rawID := taskIDs[0]
	parsed := git.ParseTaskID(rawID)
	taskID := parsed.ID
	isCustomID := parsed.IsCustomID

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	qs := cmdutil.CustomIDTaskQueryWithSubtasks(cfg, isCustomID)

	ctx := context.Background()
	task, err := apiv2.GetTaskLocal(ctx, client, taskID, qs)
	if err != nil {
		return fmt.Errorf("failed to fetch task %s: %w", taskID, err)
	}

	// Fetch markdown description and subtasks (the standard GetTask doesn't include these).
	var extras taskWithExtras
	extrasPath := fmt.Sprintf("task/%s/?include_markdown_description=true&include_subtasks=true", task.ID)
	if err := apiv2.Do(ctx, client, "GET", extrasPath, nil, &extras); err == nil {
		if extras.MarkdownDescription != "" {
			task.MarkdownDescription = extras.MarkdownDescription
		}
	}

	subtasks := extras.Subtasks
	if opts.recursive && len(subtasks) > 0 {
		fetchSubtasksRecursive(ctx, client, subtasks, ios)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, struct {
			*clickup.Task
			Subtasks []subtaskInfo `json:"subtasks"`
		}{task, subtasks})
	}

	return printTaskView(f, task, subtasks)
}

// runViewBulk fetches multiple tasks concurrently with bounded parallelism.
func runViewBulk(f *cmdutil.Factory, opts *viewOptions, rawIDs []string) error {
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

	type taskResult struct {
		Task     *clickup.Task `json:"task"`
		Subtasks []subtaskInfo `json:"subtasks"`
		Err      error         `json:"-"`
		ID       string        `json:"-"`
	}

	results := make([]taskResult, len(rawIDs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // max 10 concurrent requests

	ctx := context.Background()

	for i, rawID := range rawIDs {
		wg.Add(1)
		go func(idx int, raw string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			parsed := git.ParseTaskID(raw)
			isCustomID := parsed.IsCustomID
			taskID := parsed.ID

			results[idx].ID = taskID

			qs := cmdutil.CustomIDTaskQueryWithSubtasks(cfg, isCustomID)
			task, err := apiv2.GetTaskLocal(ctx, client, taskID, qs)
			if err != nil {
				results[idx].Err = err
				return
			}

			// Fetch subtasks.
			var extras taskWithExtras
			extrasPath := fmt.Sprintf("task/%s/?include_markdown_description=true&include_subtasks=true", task.ID)
			if err := apiv2.Do(ctx, client, "GET", extrasPath, nil, &extras); err == nil {
				if extras.MarkdownDescription != "" {
					task.MarkdownDescription = extras.MarkdownDescription
				}
			}

			results[idx].Task = task
			results[idx].Subtasks = extras.Subtasks
		}(i, rawID)
	}

	wg.Wait()

	// Report errors to stderr, collect successful results.
	type taskOutput struct {
		*clickup.Task
		Subtasks []subtaskInfo `json:"subtasks"`
	}
	var output []taskOutput
	var errCount int
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(ios.ErrOut, "%s failed to fetch %s: %v\n", cs.Red("✗"), r.ID, r.Err)
			errCount++
			continue
		}
		if opts.recursive && len(r.Subtasks) > 0 {
			fetchSubtasksRecursive(ctx, client, r.Subtasks, ios)
		}
		output = append(output, taskOutput{r.Task, r.Subtasks})
	}

	if len(output) > 0 {
		fmt.Fprintf(ios.ErrOut, "Fetched %d/%d tasks\n", len(output), len(rawIDs))
	}

	return opts.jsonFlags.OutputJSON(ios.Out, output)
}

func printTaskView(f *cmdutil.Factory, task *clickup.Task, subtasks []subtaskInfo) error {
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

	// Subtasks
	if len(subtasks) > 0 {
		fmt.Fprintf(out, "\n%s\n", cs.Bold("Subtasks:"))
		for _, st := range subtasks {
			id := st.ID
			if st.CustomID != "" {
				id = st.CustomID
			}
			statusText := st.Status.Status
			statusColorFn := cs.StatusColor(strings.ToLower(statusText))
			var assignees string
			if len(st.Assignees) > 0 {
				names := make([]string, 0, len(st.Assignees))
				for _, a := range st.Assignees {
					names = append(names, a.Username)
				}
				assignees = fmt.Sprintf(" (%s)", strings.Join(names, ", "))
			}
			var dates string
			if st.DueDate != nil {
				if dt := st.DueDate.Time(); dt != nil {
					dates = cs.Gray(" due:" + dt.Format("2006-01-02"))
				}
			}
			if dates == "" && st.StartDate != "" {
				if t, err := parseUnixMillis(st.StartDate); err == nil {
					dates = cs.Gray(" start:" + t.Format("2006-01-02"))
				}
			}
			fmt.Fprintf(out, "  #%s %s %s%s%s\n", id, statusColorFn(statusText), st.Name, assignees, dates)
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
	fmt.Fprintf(out, "  %s  clickup comment add %s \"@user text\" (supports @mentions)\n", cs.Gray("Comment:"), id)
	fmt.Fprintf(out, "  %s  clickup link pr --task %s\n", cs.Gray("Link PR:"), id)
	fmt.Fprintf(out, "  %s  clickup task view %s --json\n", cs.Gray("JSON:"), id)

	return nil
}

// fetchSubtasksRecursive walks a slice of subtasks, fetching each one's children
// concurrently and recursing until the tree is fully expanded.
func fetchSubtasksRecursive(ctx context.Context, client *api.Client, subtasks []subtaskInfo, ios *iostreams.IOStreams) {
	if len(subtasks) == 0 {
		return
	}

	cs := ios.ColorScheme()
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	var mu sync.Mutex
	var withChildren []*subtaskInfo

	for i := range subtasks {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var extras taskWithExtras
			extrasPath := fmt.Sprintf("task/%s/?include_subtasks=true", subtasks[idx].ID)
			if err := apiv2.Do(ctx, client, "GET", extrasPath, nil, &extras); err != nil {
				fmt.Fprintf(ios.ErrOut, "%s failed to fetch subtasks for %s: %v\n", cs.Yellow("!"), subtasks[idx].ID, err)
				return
			}
			if len(extras.Subtasks) > 0 {
				subtasks[idx].Subtasks = extras.Subtasks
				mu.Lock()
				withChildren = append(withChildren, &subtasks[idx])
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Recurse into children that themselves have subtasks.
	for _, st := range withChildren {
		fetchSubtasksRecursive(ctx, client, st.Subtasks, ios)
	}
}

// findTaskViaPR detects the current branch's PR URL and searches task descriptions
// for it using progressive drill-down. Returns (taskID, isCustomID, prNumber).
func findTaskViaPR(f *cmdutil.Factory, ios *iostreams.IOStreams) (string, bool, int) {
	prURL, prNum := detectPRInfo()
	if prURL == "" {
		return "", false, 0
	}

	client, err := f.ApiClient()
	if err != nil {
		return "", false, 0
	}

	cfg, err := f.Config()
	if err != nil {
		return "", false, 0
	}

	teamID := cfg.Workspace
	if teamID == "" {
		return "", false, 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Progressive drill-down: sprint → user → space → workspace.
	type level struct {
		label       string
		extraParams string
		maxPages    int
	}

	var levels []level

	if cfg.SprintFolder != "" {
		listID, err := cmdutil.ResolveCurrentSprintListID(ctx, client, cfg.SprintFolder)
		if err == nil && listID != "" {
			levels = append(levels, level{"sprint", "list_ids[]=" + listID, 1})
		}
	}

	if userID, err := cmdutil.GetCurrentUserID(client); err == nil {
		levels = append(levels, level{"your tasks", fmt.Sprintf("assignees[]=%d", userID), 1})
	}

	if cfg.Space != "" {
		levels = append(levels, level{"space", "space_ids[]=" + cfg.Space, 3})
	}

	levels = append(levels, level{"workspace", "", 10})

	for _, lvl := range levels {
		if ctx.Err() != nil {
			break
		}
		fmt.Fprintf(ios.ErrOut, "  searching %s for PR #%d...\n", lvl.label, prNum)
		for page := 0; page < lvl.maxPages; page++ {
			tasks, err := apiv2.FetchTeamTasks(ctx, client, teamID, page, lvl.extraParams)
			if err != nil || len(tasks) == 0 {
				break
			}
			for _, t := range tasks {
				if strings.Contains(t.Description, prURL) {
					id := t.ID
					isCustom := false
					if t.CustomID != "" {
						id = t.CustomID
						isCustom = true
					}
					return id, isCustom, prNum
				}
			}
		}
	}

	return "", false, 0
}

// detectPRInfo uses the GitHub CLI to detect the current branch's PR URL and number.
// Returns ("", 0) on any error.
func detectPRInfo() (string, int) {
	out, err := exec.Command("gh", "pr", "view", "--json", "url,number").Output()
	if err != nil {
		return "", 0
	}
	var pr struct {
		URL    string `json:"url"`
		Number int    `json:"number"`
	}
	if json.Unmarshal(out, &pr) != nil {
		return "", 0
	}
	return pr.URL, pr.Number
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
