package task

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdTime returns the parent command for time tracking subcommands.
func NewCmdTime(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time <command>",
		Short: "Track time on ClickUp tasks",
		Long:  "Log time and view time entries for ClickUp tasks.",
	}

	cmd.AddCommand(NewCmdTimeLog(f))
	cmd.AddCommand(NewCmdTimeList(f))

	return cmd
}

// --- time log ---

type timeLogOptions struct {
	taskID      string
	duration    string
	description string
	date        string
	billable    bool
}

// NewCmdTimeLog returns a command to log a time entry on a task.
func NewCmdTimeLog(f *cmdutil.Factory) *cobra.Command {
	opts := &timeLogOptions{}

	cmd := &cobra.Command{
		Use:   "log [<task-id>]",
		Short: "Log time to a task",
		Long: `Log a time entry against a ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name.`,
		Example: `  # Log 2 hours to a specific task
  clickup task time log 86a3xrwkp --duration 2h

  # Log 1h30m with a description
  clickup task time log --duration 1h30m --description "Implemented auth flow"

  # Log time for a specific date
  clickup task time log 86a3xrwkp --duration 45m --date 2025-01-15

  # Log billable time
  clickup task time log --duration 3h --billable`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.taskID = args[0]
			}
			if opts.duration == "" {
				return fmt.Errorf("required flag --duration not set")
			}
			return runTimeLog(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.duration, "duration", "", "Duration to log (e.g. \"2h\", \"30m\", \"1h30m\")")
	cmd.Flags().StringVar(&opts.description, "description", "", "Description of work done")
	cmd.Flags().StringVar(&opts.date, "date", "", "Date of the work (YYYY-MM-DD, default today)")
	cmd.Flags().BoolVar(&opts.billable, "billable", false, "Mark time entry as billable")

	_ = cmd.MarkFlagRequired("duration")

	return cmd
}

func runTimeLog(f *cmdutil.Factory, opts *timeLogOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	taskID := opts.taskID

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
	} else {
		parsed := git.ParseTaskID(taskID)
		taskID = parsed.ID
	}

	// Parse duration.
	d, err := time.ParseDuration(opts.duration)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", opts.duration, err)
	}
	durationMs := d.Milliseconds()

	// Parse date (default to today).
	var startDate time.Time
	if opts.date != "" {
		startDate, err = time.Parse("2006-01-02", opts.date)
		if err != nil {
			return fmt.Errorf("invalid date %q (expected YYYY-MM-DD): %w", opts.date, err)
		}
	} else {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	// Set start to 9:00 AM on the given date.
	startTime := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 9, 0, 0, 0, startDate.Location())
	startMs := startTime.UnixMilli()

	// Get workspace (team) ID from config.
	cfg, err := f.Config()
	if err != nil {
		return err
	}
	teamID := cfg.Workspace
	if teamID == "" {
		return fmt.Errorf("workspace not configured. Run 'clickup config set workspace <id>' first")
	}

	// Build request body.
	body := map[string]interface{}{
		"description": opts.description,
		"start":       startMs,
		"duration":    durationMs,
		"billable":    opts.billable,
		"tid":         taskID,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("https://api.clickup.com/api/v2/team/%s/time_entries", teamID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	fmt.Fprintf(ios.Out, "%s Logged %s to task %s\n",
		cs.Green("✓"),
		cs.Bold(formatDuration(strconv.FormatInt(durationMs, 10))),
		cs.Bold(taskID),
	)

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task time list %s\n", cs.Gray("Entries:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task time list %s --json\n", cs.Gray("JSON:"), taskID)

	return nil
}

// --- time list ---

type timeListOptions struct {
	taskID    string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdTimeList returns a command to list time entries for a task.
func NewCmdTimeList(f *cmdutil.Factory) *cobra.Command {
	opts := &timeListOptions{}

	cmd := &cobra.Command{
		Use:   "list [<task-id>]",
		Short: "View time entries for a task",
		Long: `Display time entries logged against a ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name.`,
		Example: `  # List time entries for a specific task
  clickup task time list 86a3xrwkp

  # Auto-detect task from git branch
  clickup task time list

  # Output as JSON
  clickup task time list 86a3xrwkp --json

  # Filter with jq
  clickup task time list 86a3xrwkp --jq '.[] | .duration'`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.taskID = args[0]
			}
			return runTimeList(f, opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

type timeEntryResponse struct {
	Data []timeEntry `json:"data"`
}

type timeEntry struct {
	ID          string `json:"id"`
	Duration    string `json:"duration"`
	Description string `json:"description"`
	Start       string `json:"start"`
	End         string `json:"end"`
	User        struct {
		Username string `json:"username"`
	} `json:"user"`
	Billable bool `json:"billable"`
}

func runTimeList(f *cmdutil.Factory, opts *timeListOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	taskID := opts.taskID

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
	} else {
		parsed := git.ParseTaskID(taskID)
		taskID = parsed.ID
	}

	// Get workspace (team) ID from config.
	cfg, err := f.Config()
	if err != nil {
		return err
	}
	teamID := cfg.Workspace
	if teamID == "" {
		return fmt.Errorf("workspace not configured. Run 'clickup config set workspace <id>' first")
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.clickup.com/api/v2/team/%s/time_entries?task_id=%s", teamID, taskID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result timeEntryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, result.Data)
	}

	if len(result.Data) == 0 {
		fmt.Fprintln(ios.ErrOut, "No time entries found.")
		return nil
	}

	return printTimeEntryTable(f, result.Data, taskID)
}

func printTimeEntryTable(f *cmdutil.Factory, entries []timeEntry, taskID string) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	tp := tableprinter.New(ios)

	// Header row.
	tp.AddField(cs.Bold("DATE"))
	tp.AddField(cs.Bold("USER"))
	tp.AddField(cs.Bold("DURATION"))
	tp.AddField(cs.Bold("DESCRIPTION"))
	tp.AddField(cs.Bold("BILLABLE"))
	tp.EndRow()

	tp.SetTruncateColumn(3) // Truncate description column if table is too wide.

	for _, e := range entries {
		// Convert start ms to date.
		dateStr := e.Start
		if t, err := parseUnixMillis(e.Start); err == nil {
			dateStr = t.Format("2006-01-02")
		}
		tp.AddField(dateStr)

		tp.AddField(e.User.Username)
		tp.AddField(formatDuration(e.Duration))
		tp.AddField(e.Description)

		billableStr := "No"
		if e.Billable {
			billableStr = "Yes"
		}
		tp.AddField(billableStr)

		tp.EndRow()
	}

	if err := tp.Render(); err != nil {
		return err
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task time log %s --duration <dur>\n", cs.Gray("Log:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task time list %s --json\n", cs.Gray("JSON:"), taskID)

	return nil
}

// formatDuration converts a duration in milliseconds (as a string) to a human-readable format.
func formatDuration(msStr string) string {
	ms, err := strconv.ParseInt(msStr, 10, 64)
	if err != nil {
		return msStr
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
	return fmt.Sprintf("%dm", m)
}
