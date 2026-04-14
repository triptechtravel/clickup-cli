package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	clickupv2 "github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
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
	cmd.AddCommand(NewCmdTimeDelete(f))
	cmd.AddCommand(NewCmdTimeStart(f))
	cmd.AddCommand(NewCmdTimeStop(f))
	cmd.AddCommand(NewCmdTimeRunning(f))

	return cmd
}

// --- time log ---

type timeLogOptions struct {
	taskID      string
	duration    string
	description string
	date        string
	assignee    string
	billable    bool
	fromFile    string
}

// NewCmdTimeLog returns a command to log a time entry on a task.
func NewCmdTimeLog(f *cmdutil.Factory) *cobra.Command {
	opts := &timeLogOptions{}

	cmd := &cobra.Command{
		Use:   "log [<task-id>]",
		Short: "Log time to a task",
		Long: `Log a time entry against a ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name.

Use --from-file to bulk log time entries from a JSON file. The file should
contain an array of objects with task_id, duration, and optionally date,
description, assignee, and billable fields.`,
		Example: `  # Log 2 hours to a specific task
  clickup task time log 86a3xrwkp --duration 2h

  # Log 1h30m with a description
  clickup task time log --duration 1h30m --description "Implemented auth flow"

  # Log time for a specific date
  clickup task time log 86a3xrwkp --duration 45m --date 2025-01-15

  # Log billable time
  clickup task time log --duration 3h --billable

  # Log time for another team member
  clickup task time log 86a3xrwkp --duration 2h --assignee 54874661

  # Bulk log from a JSON file
  clickup task time log --from-file entries.json`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.fromFile != "" {
				return runBulkTimeLog(f, opts)
			}
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
	cmd.Flags().StringVar(&opts.assignee, "assignee", "", "User ID to log time for (default: current user)")
	cmd.Flags().BoolVar(&opts.billable, "billable", false, "Mark time entry as billable")
	cmd.Flags().StringVar(&opts.fromFile, "from-file", "", "Log time entries from a JSON file (array of entry objects)")

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
		startDate, err = time.ParseInLocation("2006-01-02", opts.date, time.Local)
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

	// Build typed request.
	req := &clickupv2.CreateatimeentryJSONRequest{
		Description: &opts.description,
		Start:       int(startMs),
		Duration:    int(durationMs),
		Billable:    &opts.billable,
		Tid:         &taskID,
	}
	if opts.assignee != "" {
		assigneeID, err := strconv.Atoi(opts.assignee)
		if err != nil {
			return fmt.Errorf("invalid assignee ID %q: must be a numeric user ID", opts.assignee)
		}
		req.Assignee = &assigneeID
	}
	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	// TODO: swap to generated wrapper — Createatimeentry response type lacks
	// the data.id wrapper the real API returns, so we keep apiv2.Do with a
	// local struct to extract the entry ID.
	ctx := context.Background()
	var logResult struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := apiv2.Do(ctx, client, "POST", fmt.Sprintf("team/%s/time_entries", teamID), req, &logResult); err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	entryID := logResult.Data.ID

	fmt.Fprintf(ios.Out, "%s Logged %s to task %s",
		cs.Green("✓"),
		cs.Bold(formatDuration(strconv.FormatInt(durationMs, 10))),
		cs.Bold(taskID),
	)
	if entryID != "" {
		fmt.Fprintf(ios.Out, " %s", cs.Gray("(entry "+entryID+")"))
	}
	fmt.Fprintln(ios.Out)

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task time list %s\n", cs.Gray("Entries:"), taskID)
	if entryID != "" {
		fmt.Fprintf(ios.Out, "  %s  clickup task time delete %s\n", cs.Gray("Delete:"), entryID)
	}
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task time list %s --json\n", cs.Gray("JSON:"), taskID)

	return nil
}

// --- time log bulk ---

type timeLogFileEntry struct {
	TaskID      string `json:"task_id"`
	Duration    string `json:"duration"`
	Description string `json:"description"`
	Date        string `json:"date"`
	Assignee    string `json:"assignee"`
	Billable    bool   `json:"billable"`
}

func runBulkTimeLog(f *cmdutil.Factory, opts *timeLogOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	data, err := os.ReadFile(opts.fromFile)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", opts.fromFile, err)
	}

	var entries []timeLogFileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no entries found in %s", opts.fromFile)
	}

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

	total := len(entries)
	var logged int

	for i, entry := range entries {
		if entry.TaskID == "" {
			fmt.Fprintf(ios.ErrOut, "%s (%d/%d) Skipped: task_id is empty\n", cs.Yellow("!"), i+1, total)
			continue
		}
		if entry.Duration == "" {
			fmt.Fprintf(ios.ErrOut, "%s (%d/%d) Skipped: duration is empty for task %s\n", cs.Yellow("!"), i+1, total, entry.TaskID)
			continue
		}

		d, err := time.ParseDuration(entry.Duration)
		if err != nil {
			fmt.Fprintf(ios.ErrOut, "%s (%d/%d) Skipped: invalid duration %q for task %s: %v\n",
				cs.Red("✗"), i+1, total, entry.Duration, entry.TaskID, err)
			continue
		}
		durationMs := d.Milliseconds()

		// Parse date (default to today).
		var startDate time.Time
		if entry.Date != "" {
			startDate, err = time.ParseInLocation("2006-01-02", entry.Date, time.Local)
			if err != nil {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) Skipped: invalid date %q for task %s\n",
					cs.Red("✗"), i+1, total, entry.Date, entry.TaskID)
				continue
			}
		} else {
			now := time.Now()
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		}

		startTime := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 9, 0, 0, 0, startDate.Location())
		startMs := startTime.UnixMilli()

		parsed := git.ParseTaskID(entry.TaskID)
		taskID := parsed.ID

		req := &clickupv2.CreateatimeentryJSONRequest{
			Description: &entry.Description,
			Start:       int(startMs),
			Duration:    int(durationMs),
			Billable:    &entry.Billable,
			Tid:         &taskID,
		}

		// Use entry-level assignee, fall back to flag-level.
		assigneeStr := entry.Assignee
		if assigneeStr == "" {
			assigneeStr = opts.assignee
		}
		if assigneeStr != "" {
			assigneeID, err := strconv.Atoi(assigneeStr)
			if err != nil {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) Skipped: invalid assignee %q for task %s\n",
					cs.Red("✗"), i+1, total, assigneeStr, entry.TaskID)
				continue
			}
			req.Assignee = &assigneeID
		}

		ctx := context.Background()
		if _, err := apiv2.Createatimeentry(ctx, client, teamID, req); err != nil {
			fmt.Fprintf(ios.ErrOut, "%s (%d/%d) Failed: task %s: %v\n",
				cs.Red("✗"), i+1, total, entry.TaskID, err)
			continue
		}

		logged++
		fmt.Fprintf(ios.Out, "%s (%d/%d) Logged %s to task %s",
			cs.Green("✓"), i+1, total,
			formatDuration(strconv.FormatInt(durationMs, 10)),
			entry.TaskID,
		)
		if entry.Description != "" {
			desc := entry.Description
			if len(desc) > 50 {
				desc = desc[:50] + "..."
			}
			fmt.Fprintf(ios.Out, " %s", cs.Gray(desc))
		}
		fmt.Fprintln(ios.Out)
	}

	fmt.Fprintf(ios.Out, "\n%s Logged %d/%d entries\n", cs.Green("!"), logged, total)
	return nil
}

// --- time list ---

type timeListOptions struct {
	taskID      string
	startDate   string
	endDate     string
	assignee    string
	tags        []string
	includeTags bool
	jsonFlags   cmdutil.JSONFlags
}

// NewCmdTimeList returns a command to list time entries for a task or date range.
func NewCmdTimeList(f *cmdutil.Factory) *cobra.Command {
	opts := &timeListOptions{}

	cmd := &cobra.Command{
		Use:   "list [<task-id>]",
		Short: "View time entries for a task or date range",
		Long: `Display time entries logged against a ClickUp task, or query all entries
across tasks for a date range (timesheet mode).

Per-task mode (default): Shows entries for a single task. If no task ID is
provided, the CLI auto-detects it from the current git branch name.

Timesheet mode: When --start-date and --end-date are provided, shows all
time entries across tasks for the given date range. By default filters to
the current user; use --assignee to change.`,
		Example: `  # List time entries for a specific task
  clickup task time list 86a3xrwkp

  # Auto-detect task from git branch
  clickup task time list

  # Timesheet: all your entries for a month
  clickup task time list --start-date 2026-02-01 --end-date 2026-02-28

  # Timesheet for all workspace members
  clickup task time list --start-date 2026-02-01 --end-date 2026-02-28 --assignee all

  # Timesheet for a specific user
  clickup task time list --start-date 2026-02-01 --end-date 2026-02-28 --assignee 54695018

  # Timesheet for multiple users (fetched concurrently)
  clickup task time list --start-date 2026-03-01 --end-date 2026-03-31 --assignee 48884897,54874661,54874662

  # Output as JSON
  clickup task time list 86a3xrwkp --json

  # Filter with jq
  clickup task time list --start-date 2026-02-01 --end-date 2026-02-28 --jq '[.[] | {task: .task.name, hrs: (.duration | tonumber / 3600000)}]'`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.taskID = args[0]
			}
			return runTimeList(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.startDate, "start-date", "", "Start date for timesheet mode (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.endDate, "end-date", "", "End date for timesheet mode (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.assignee, "assignee", "", `Filter by user ID(s) — comma-separated, or "all" for everyone (default: current user)`)
	cmd.Flags().StringSliceVar(&opts.tags, "tag", nil, `Filter by task tag(s) — comma-separated or repeated (OR logic, timesheet mode only)`)
	cmd.Flags().BoolVar(&opts.includeTags, "include-tags", false, "Include task tags in JSON output (fetches concurrently, timesheet mode only)")
	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

type timeEntryResponse struct {
	Data []timeEntry `json:"data"`
}

type timeEntryTask struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type timeEntryTaskLocation struct {
	ListID string `json:"list_id"`
}

type timeEntry struct {
	ID          string         `json:"id"`
	Duration    string         `json:"duration"`
	Description string         `json:"description"`
	Start       string         `json:"start"`
	End         string         `json:"end"`
	User        struct {
		Username string `json:"username"`
	} `json:"user"`
	Billable     bool                  `json:"billable"`
	Task         *timeEntryTask        `json:"task,omitempty"`
	TaskLocation *timeEntryTaskLocation `json:"task_location,omitempty"`
}

func runTimeList(f *cmdutil.Factory, opts *timeListOptions) error {
	timesheetMode := opts.startDate != "" || opts.endDate != ""

	if timesheetMode {
		return runTimeListTimesheet(f, opts)
	}
	return runTimeListPerTask(f, opts)
}

func runTimeListPerTask(f *cmdutil.Factory, opts *timeListOptions) error {
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

	// TODO: swap to generated wrapper — Gettimeentrieswithinadaterange response
	// type has User as any (not struct with Username), Task as struct (not pointer),
	// and lacks TaskLocation, so the local timeEntry struct is needed for rendering.
	ctx := context.Background()
	var result timeEntryResponse
	if err := apiv2.Do(ctx, client, "GET", fmt.Sprintf("team/%s/time_entries?task_id=%s", teamID, taskID), nil, &result); err != nil {
		return fmt.Errorf("request failed: %w", err)
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

func runTimeListTimesheet(f *cmdutil.Factory, opts *timeListOptions) error {
	ios := f.IOStreams

	// Validate both dates are provided.
	if opts.startDate == "" || opts.endDate == "" {
		return fmt.Errorf("both --start-date and --end-date are required for timesheet mode")
	}

	// Parse dates.
	startTime, err := time.ParseInLocation("2006-01-02", opts.startDate, time.Local)
	if err != nil {
		return fmt.Errorf("invalid --start-date %q (expected YYYY-MM-DD): %w", opts.startDate, err)
	}
	// End date is inclusive: set to end of day.
	endTime, err := time.ParseInLocation("2006-01-02", opts.endDate, time.Local)
	if err != nil {
		return fmt.Errorf("invalid --end-date %q (expected YYYY-MM-DD): %w", opts.endDate, err)
	}
	endTime = endTime.Add(24*time.Hour - time.Millisecond)

	startMs := startTime.UnixMilli()
	endMs := endTime.UnixMilli()

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

	// Resolve assignee IDs.
	var assigneeIDs []string
	if opts.assignee == "" || opts.assignee == "me" {
		userID, err := cmdutil.GetCurrentUserID(client)
		if err != nil {
			return fmt.Errorf("could not determine current user: %w", err)
		}
		assigneeIDs = []string{fmt.Sprintf("%d", userID)}
	} else if opts.assignee == "all" {
		assigneeIDs = nil // no filter
	} else {
		assigneeIDs = strings.Split(opts.assignee, ",")
	}

	// Fetch time entries — one request per assignee (or one unfiltered for "all").
	ctx := context.Background()
	var result timeEntryResponse
	if len(assigneeIDs) <= 1 {
		// Single assignee or "all" — one API call.
		path := fmt.Sprintf("team/%s/time_entries?start_date=%d&end_date=%d", teamID, startMs, endMs)
		if len(assigneeIDs) == 1 {
			path += fmt.Sprintf("&assignee=%s", assigneeIDs[0])
		}
		entries, err := fetchTimeEntries(ctx, client, path)
		if err != nil {
			return err
		}
		result.Data = entries
	} else {
		// Multiple assignees — fetch concurrently with bounded parallelism.
		type fetchResult struct {
			entries []timeEntry
			err     error
		}
		results := make([]fetchResult, len(assigneeIDs))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 5)

		for i, aid := range assigneeIDs {
			wg.Add(1)
			go func(idx int, assignee string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				path := fmt.Sprintf("team/%s/time_entries?start_date=%d&end_date=%d&assignee=%s",
					teamID, startMs, endMs, assignee)
				entries, err := fetchTimeEntries(ctx, client, path)
				results[idx] = fetchResult{entries, err}
			}(i, aid)
		}
		wg.Wait()

		for _, r := range results {
			if r.err != nil {
				return r.err
			}
			result.Data = append(result.Data, r.entries...)
		}
	}

	// Filter by tag if requested.
	if len(opts.tags) > 0 {
		filtered, err := filterTimeEntriesByTags(client, result.Data, opts.tags)
		if err != nil {
			return fmt.Errorf("failed to filter by tag: %w", err)
		}
		result.Data = filtered
	}

	if opts.jsonFlags.WantsJSON() {
		// Enrich with tags if requested.
		if opts.includeTags {
			enriched, err := enrichTimeEntriesWithTags(client, result.Data)
			if err != nil {
				return fmt.Errorf("failed to fetch tags: %w", err)
			}
			return opts.jsonFlags.OutputJSON(ios.Out, enriched)
		}
		return opts.jsonFlags.OutputJSON(ios.Out, result.Data)
	}

	if len(result.Data) == 0 {
		fmt.Fprintln(ios.ErrOut, "No time entries found.")
		return nil
	}

	return printTimesheetTable(f, result.Data, opts.startDate, opts.endDate)
}

// timeEntryWithTags extends timeEntry with task tag names.
type timeEntryWithTags struct {
	timeEntry
	Tags []string `json:"tags"`
}

// enrichTimeEntriesWithTags fetches tags for all unique tasks concurrently
// and returns enriched entries with a "tags" field.
func enrichTimeEntriesWithTags(client *api.Client, entries []timeEntry) ([]timeEntryWithTags, error) {
	// Collect unique task IDs.
	taskIDs := make(map[string]bool)
	for _, e := range entries {
		if e.Task != nil {
			taskIDs[e.Task.ID] = true
		}
	}

	// Fetch tags concurrently with bounded parallelism.
	type tagResult struct {
		taskID string
		tags   []string
		err    error
	}

	results := make(chan tagResult, len(taskIDs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	ctx := context.Background()

	for tid := range taskIDs {
		wg.Add(1)
		go func(taskID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			task, err := apiv2.GetTaskLocal(ctx, client, taskID, "")
			if err != nil {
				results <- tagResult{taskID, nil, err}
				return
			}
			var tags []string
			for _, t := range task.Tags {
				tags = append(tags, t.Name)
			}
			results <- tagResult{taskID, tags, nil}
		}(tid)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Build lookup.
	tagLookup := make(map[string][]string)
	for r := range results {
		if r.err != nil {
			// Non-fatal: skip tasks we can't fetch.
			continue
		}
		tagLookup[r.taskID] = r.tags
	}

	// Enrich entries.
	enriched := make([]timeEntryWithTags, len(entries))
	for i, e := range entries {
		tags := []string{}
		if e.Task != nil {
			if t, ok := tagLookup[e.Task.ID]; ok {
				tags = t
			}
		}
		enriched[i] = timeEntryWithTags{
			timeEntry: e,
			Tags:      tags,
		}
	}

	return enriched, nil
}

// fetchTimeEntries performs a single GET request and returns the time entries.
//
// TODO: swap to generated wrapper — Gettimeentrieswithinadaterange response
// type has User as any (not struct with Username), Task as struct (not pointer),
// and lacks TaskLocation, so the local timeEntry struct is needed for rendering.
func fetchTimeEntries(ctx context.Context, client *api.Client, path string) ([]timeEntry, error) {
	var result timeEntryResponse
	if err := apiv2.Do(ctx, client, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return result.Data, nil
}

// filterTimeEntriesByTags filters time entries to only those whose task has at
// least one of the specified tags. It extracts unique list IDs from the entries'
// task_location, fetches tasks with the tag filter from each list (handling
// pagination), and then keeps only entries whose task ID is in the qualifying set.
func filterTimeEntriesByTags(client *api.Client, entries []timeEntry, tags []string) ([]timeEntry, error) {
	// Collect unique list IDs from time entries.
	listIDs := make(map[string]bool)
	for _, e := range entries {
		if e.TaskLocation != nil && e.TaskLocation.ListID != "" {
			listIDs[e.TaskLocation.ListID] = true
		}
	}

	if len(listIDs) == 0 {
		return nil, nil
	}

	// For each list, fetch all tasks matching the tags (with pagination).
	qualifyingTaskIDs := make(map[string]bool)
	ctx := context.Background()

	for listID := range listIDs {
		page := 0
		for {
			q := "?include_closed=true"
			q += fmt.Sprintf("&page=%d", page)
			for _, tag := range tags {
				q += "&tags[]=" + tag
			}
			tasks, err := apiv2.GetTasksLocal(ctx, client, listID, q)
			if err != nil {
				// If we get a permission error for a list, skip it rather than failing.
				if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "ECODE") {
					break
				}
				return nil, fmt.Errorf("failed to get tasks for list %s: %w", listID, err)
			}
			for _, t := range tasks {
				qualifyingTaskIDs[t.ID] = true
			}
			// ClickUp returns max 100 tasks per page; if fewer, we're done.
			if len(tasks) < 100 {
				break
			}
			page++
		}
	}

	// Filter entries to only those with a qualifying task ID.
	var filtered []timeEntry
	for _, e := range entries {
		if e.Task != nil && qualifyingTaskIDs[e.Task.ID] {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

func printTimesheetTable(f *cmdutil.Factory, entries []timeEntry, startDate, endDate string) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	tp := tableprinter.New(ios)

	fmt.Fprintf(ios.Out, "%s  %s to %s\n\n",
		cs.Bold("Timesheet"),
		cs.Cyan(startDate),
		cs.Cyan(endDate),
	)

	// Header row.
	tp.AddField(cs.Bold("DATE"))
	tp.AddField(cs.Bold("TASK"))
	tp.AddField(cs.Bold("USER"))
	tp.AddField(cs.Bold("DURATION"))
	tp.AddField(cs.Bold("DESCRIPTION"))
	tp.EndRow()

	tp.SetTruncateColumn(4) // Truncate description column if table is too wide.

	var totalMs int64
	for _, e := range entries {
		// Convert start ms to date.
		dateStr := e.Start
		if t, err := parseUnixMillis(e.Start); err == nil {
			dateStr = t.Format("2006-01-02")
		}
		tp.AddField(dateStr)

		taskName := ""
		if e.Task != nil {
			taskName = e.Task.Name
		}
		tp.AddField(taskName)

		tp.AddField(e.User.Username)
		tp.AddField(formatDuration(e.Duration))
		tp.AddField(e.Description)

		tp.EndRow()

		if ms, err := strconv.ParseInt(e.Duration, 10, 64); err == nil {
			totalMs += ms
		}
	}

	if err := tp.Render(); err != nil {
		return err
	}

	// Totals summary.
	fmt.Fprintln(ios.Out)
	fmt.Fprintf(ios.Out, "%s %s across %d entries\n",
		cs.Bold("Total:"),
		cs.Green(formatDuration(strconv.FormatInt(totalMs, 10))),
		len(entries),
	)

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task time list --start-date %s --end-date %s --json\n", cs.Gray("JSON:"), startDate, endDate)
	fmt.Fprintf(ios.Out, "  %s  clickup task time log <task-id> --duration <dur>\n", cs.Gray("Log:"))

	return nil
}

func printTimeEntryTable(f *cmdutil.Factory, entries []timeEntry, taskID string) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	tp := tableprinter.New(ios)

	// Header row.
	tp.AddField(cs.Bold("ID"))
	tp.AddField(cs.Bold("DATE"))
	tp.AddField(cs.Bold("USER"))
	tp.AddField(cs.Bold("DURATION"))
	tp.AddField(cs.Bold("DESCRIPTION"))
	tp.AddField(cs.Bold("BILLABLE"))
	tp.EndRow()

	tp.SetTruncateColumn(4) // Truncate description column if table is too wide.

	for _, e := range entries {
		tp.AddField(e.ID)

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
	fmt.Fprintf(ios.Out, "  %s  clickup task time delete <entry-id>\n", cs.Gray("Delete:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task time list %s --json\n", cs.Gray("JSON:"), taskID)

	return nil
}

// --- time delete ---

type timeDeleteOptions struct {
	factory *cmdutil.Factory
	entryID string
	confirm bool
}

// NewCmdTimeDelete returns a command to delete a time entry.
func NewCmdTimeDelete(f *cmdutil.Factory) *cobra.Command {
	opts := &timeDeleteOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "delete <entry-id>",
		Short: "Delete a time entry",
		Long: `Delete a time entry from ClickUp.

ENTRY_ID is required. Find entry IDs with 'clickup task time list TASK_ID'.
Use --yes to skip the confirmation prompt.`,
		Example: `  # Delete a time entry (with confirmation)
  clickup task time delete 1234567890

  # Delete without confirmation
  clickup task time delete 1234567890 --yes

  # Find entry IDs first
  clickup task time list 86a3xrwkp`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.entryID = args[0]
			return runTimeDelete(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.confirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runTimeDelete(opts *timeDeleteOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	if !opts.confirm && ios.IsTerminal() {
		p := prompter.New(ios)
		ok, err := p.Confirm(fmt.Sprintf("Delete time entry %s?", opts.entryID), false)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(ios.ErrOut, "Cancelled.")
			return nil
		}
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return err
	}
	teamID := cfg.Workspace
	if teamID == "" {
		return fmt.Errorf("workspace not configured. Run 'clickup config set workspace <id>' first")
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	if _, err := apiv2.DeleteatimeEntry(ctx, client, teamID, opts.entryID); err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	fmt.Fprintf(ios.Out, "%s Time entry %s deleted\n", cs.Green("!"), cs.Bold(opts.entryID))
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
