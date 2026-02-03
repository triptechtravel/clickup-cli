package sprint

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdSprintCurrent returns the sprint current command.
func NewCmdSprintCurrent(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags
	var folderID string

	cmd := &cobra.Command{
		Use:   "current",
		Short: "Show current sprint tasks",
		Long: `Show tasks in the currently active sprint.

Finds the sprint whose dates contain today, then lists
all tasks grouped by status with assignees, priorities,
and linked GitHub branches.`,
		Example: `  # Show current sprint tasks
  clickup sprint current

  # Specify a sprint folder
  clickup sprint current --folder 132693664

  # JSON output
  clickup sprint current --json`,
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSprintCurrent(f, folderID, &jsonFlags)
		},
	}

	cmd.Flags().StringVar(&folderID, "folder", "", "Sprint folder ID (auto-detected if not set)")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}

type sprintTaskEntry struct {
	ID           string `json:"id"`
	CustomID     string `json:"custom_id,omitempty"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Assignee     string `json:"assignee"`
	Priority     string `json:"priority"`
	DueDate      string `json:"due_date,omitempty"`
	Points       string `json:"points,omitempty"`
	TimeEstimate string `json:"time_estimate,omitempty"`
	TimeSpent    string `json:"time_spent,omitempty"`
	ListID       string `json:"list_id"`
	SprintName   string `json:"sprint_name"`
}

func runSprintCurrent(f *cmdutil.Factory, folderID string, jsonFlags *cmdutil.JSONFlags) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	ctx := context.Background()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	// Resolve sprint folder (same logic as sprint list).
	if folderID != "" {
		if cfg.SprintFolder != folderID {
			cfg.SprintFolder = folderID
			_ = cfg.Save()
		}
	} else {
		folderID = cfg.SprintFolder
	}
	if folderID == "" {
		spaceID := cfg.Space
		if spaceID == "" {
			return fmt.Errorf("no space configured. Run 'clickup space select' first")
		}

		folders, _, err := client.Clickup.Folders.GetFolders(ctx, spaceID, false)
		if err != nil {
			return fmt.Errorf("failed to list folders: %w", err)
		}

		var sprintFolders []clickup.Folder
		for _, folder := range folders {
			if strings.Contains(strings.ToLower(folder.Name), "sprint") &&
				!strings.Contains(strings.ToLower(folder.Name), "archive") {
				sprintFolders = append(sprintFolders, folder)
			}
		}

		if len(sprintFolders) == 0 {
			return fmt.Errorf("no sprint folders found. Use --folder to specify a folder ID")
		}

		if len(sprintFolders) == 1 {
			folderID = sprintFolders[0].ID
			cfg.SprintFolder = folderID
			_ = cfg.Save()
		} else {
			fmt.Fprintln(ios.ErrOut, "Multiple sprint folders found:")
			for _, sf := range sprintFolders {
				fmt.Fprintf(ios.ErrOut, "  %s  %s\n", sf.ID, sf.Name)
			}
			return fmt.Errorf("use --folder <id> to select one")
		}
	}

	// Find the current sprint (list with dates containing today).
	lists, _, err := client.Clickup.Lists.GetLists(ctx, folderID, false)
	if err != nil {
		return fmt.Errorf("failed to list sprints: %w", err)
	}

	now := time.Now()
	var currentList *clickup.List
	for i, l := range lists {
		start := parseMSTimestamp(l.StartDate)
		due := parseMSTimestamp(l.DueDate)
		if !start.IsZero() && !due.IsZero() && !now.Before(start) && !now.After(due) {
			currentList = &lists[i]
			break
		}
	}

	if currentList == nil {
		// Fallback: pick the most recent sprint by start date.
		var latest *clickup.List
		var latestStart time.Time
		for i, l := range lists {
			start := parseMSTimestamp(l.StartDate)
			if !start.IsZero() && start.After(latestStart) {
				latestStart = start
				latest = &lists[i]
			}
		}
		if latest != nil {
			currentList = latest
			fmt.Fprintf(ios.ErrOut, "No active sprint found. Showing most recent: %s\n", latest.Name)
		} else {
			return fmt.Errorf("no sprints with dates found in this folder")
		}
	}

	start := parseMSTimestamp(currentList.StartDate)
	due := parseMSTimestamp(currentList.DueDate)
	tc, _ := currentList.TaskCount.Int64()

	fmt.Fprintf(ios.Out, "%s  %s  %s\n\n",
		cs.Bold(currentList.Name),
		cs.Cyan(formatDateRange(start, due)),
		cs.Gray(text.Pluralize(int(tc), "task")),
	)

	// Fetch tasks in the sprint.
	taskOpts := &clickup.GetTasksOptions{
		IncludeClosed: true,
		Subtasks:      true,
	}

	var allTasks []clickup.Task
	for page := 0; ; page++ {
		taskOpts.Page = page
		tasks, _, err := client.Clickup.Tasks.GetTasks(ctx, currentList.ID, taskOpts)
		if err != nil {
			return fmt.Errorf("failed to fetch sprint tasks: %w", err)
		}
		if len(tasks) == 0 {
			break
		}
		allTasks = append(allTasks, tasks...)
	}

	if len(allTasks) == 0 {
		fmt.Fprintln(ios.Out, "No tasks in this sprint.")
		return nil
	}

	// Build entries.
	entries := make([]sprintTaskEntry, 0, len(allTasks))
	for _, t := range allTasks {
		id := t.ID
		customID := ""
		if t.CustomID != "" {
			customID = t.CustomID
			id = t.CustomID
		}

		assigneeNames := make([]string, 0, len(t.Assignees))
		for _, a := range t.Assignees {
			assigneeNames = append(assigneeNames, a.Username)
		}

		priority := t.Priority.Priority
		if priority == "" {
			priority = "-"
		}

		var dueStr string
		if t.DueDate != nil {
			dt := t.DueDate.Time()
			if dt != nil {
				dueStr = dt.Format("Jan 02")
			}
		}

		var pts string
		if p := t.Points.Value.String(); p != "" && p != "0" {
			pts = p
		}

		entries = append(entries, sprintTaskEntry{
			ID:           id,
			CustomID:     customID,
			Name:         t.Name,
			Status:       t.Status.Status,
			Assignee:     strings.Join(assigneeNames, ", "),
			Priority:     priority,
			DueDate:      dueStr,
			Points:       pts,
			TimeEstimate: formatSprintDuration(t.TimeEstimate),
			TimeSpent:    formatSprintDuration(t.TimeSpent),
			ListID:       currentList.ID,
			SprintName:   currentList.Name,
		})
	}

	if jsonFlags.WantsJSON() {
		return jsonFlags.OutputJSON(ios.Out, entries)
	}

	// Group by status for display.
	statusGroups := make(map[string][]sprintTaskEntry)
	var statusOrder []string
	for _, e := range entries {
		if _, seen := statusGroups[e.Status]; !seen {
			statusOrder = append(statusOrder, e.Status)
		}
		statusGroups[e.Status] = append(statusGroups[e.Status], e)
	}

	for _, status := range statusOrder {
		group := statusGroups[status]
		statusFn := cs.StatusColor(strings.ToLower(status))
		fmt.Fprintf(ios.Out, "%s  %s\n",
			statusFn(strings.ToUpper(status)),
			cs.Gray(strconv.Itoa(len(group))),
		)

		tp := tableprinter.New(ios)
		tp.SetTruncateColumn(1) // truncate name

		for _, e := range group {
			tp.AddField(e.ID)
			tp.AddField(e.Name)
			tp.AddField(e.Assignee)
			tp.AddField(e.Priority)
			tp.AddField(e.Points)
			tp.AddField(e.TimeEstimate)
			tp.AddField(e.TimeSpent)
			if e.DueDate != "" {
				tp.AddField(e.DueDate)
			}
			tp.EndRow()
		}
		tp.Render()
		fmt.Fprintln(ios.Out)
	}

	return nil
}

// formatSprintDuration converts milliseconds to a human-readable duration string.
func formatSprintDuration(ms int64) string {
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
