package task

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	clickupv2 "github.com/triptechtravel/clickup-cli/api/clickupv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// --- time start ---

type timeStartOptions struct {
	taskID      string
	description string
	billable    bool
	jsonFlags   cmdutil.JSONFlags
}

// NewCmdTimeStart returns a command to start a time entry timer.
func NewCmdTimeStart(f *cmdutil.Factory) *cobra.Command {
	opts := &timeStartOptions{}

	cmd := &cobra.Command{
		Use:   "start [<task-id>]",
		Short: "Start a time entry timer",
		Long: `Start a running time entry timer in ClickUp.

Optionally associate the timer with a task. If no task ID is given, a
free-running timer is started. The task ID can be auto-detected from
the current git branch.`,
		Example: `  # Start a timer on a task
  clickup task time start 86abc123

  # Start with a description
  clickup task time start 86abc123 --description "Working on auth"

  # Start a free-running timer
  clickup task time start`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.taskID = args[0]
			}
			return runTimeStart(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.description, "description", "", "Timer description")
	cmd.Flags().BoolVar(&opts.billable, "billable", false, "Mark as billable")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runTimeStart(f *cmdutil.Factory, opts *timeStartOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

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

	ctx := context.Background()

	req := &clickupv2.StartatimeEntryJSONRequest{}
	if opts.description != "" {
		req.Description = &opts.description
	}
	if opts.billable {
		req.Billable = &opts.billable
	}

	// Resolve task ID.
	taskID := opts.taskID
	if taskID != "" {
		parsed := git.ParseTaskID(taskID)
		taskID = parsed.ID
		req.Tid = &taskID
	}

	resp, err := apiv2.StartatimeEntry(ctx, client, teamID, req)
	if err != nil {
		return fmt.Errorf("failed to start timer: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, resp)
	}

	fmt.Fprintf(ios.Out, "%s Timer started", cs.Green("!"))
	if taskID != "" {
		fmt.Fprintf(ios.Out, " on task %s", cs.Bold(taskID))
	}
	if resp.Data.ID != "" {
		fmt.Fprintf(ios.Out, " %s", cs.Gray("(entry "+resp.Data.ID+")"))
	}
	fmt.Fprintln(ios.Out)

	return nil
}

// --- time stop ---

// NewCmdTimeStop returns a command to stop the running time entry timer.
func NewCmdTimeStop(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the running timer",
		Long:  `Stop the currently running time entry timer in ClickUp.`,
		Example: `  # Stop the running timer
  clickup task time stop`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTimeStop(f, &jsonFlags)
		},
	}

	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}

func runTimeStop(f *cmdutil.Factory, jsonFlags *cmdutil.JSONFlags) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

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

	ctx := context.Background()
	resp, err := apiv2.StopatimeEntry(ctx, client, teamID)
	if err != nil {
		return fmt.Errorf("failed to stop timer: %w", err)
	}

	if jsonFlags.WantsJSON() {
		return jsonFlags.OutputJSON(ios.Out, resp)
	}

	dur := formatDuration(strconv.Itoa(resp.Data.Duration))
	fmt.Fprintf(ios.Out, "%s Timer stopped — %s logged", cs.Green("!"), cs.Bold(dur))
	if resp.Data.Task.ID != "" {
		fmt.Fprintf(ios.Out, " on task %s", cs.Bold(resp.Data.Task.ID))
	}
	fmt.Fprintln(ios.Out)

	return nil
}

// --- time running ---

// NewCmdTimeRunning returns a command to show the current running timer.
func NewCmdTimeRunning(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags

	cmd := &cobra.Command{
		Use:   "running",
		Short: "Show the current running timer",
		Long:  `Display the currently running time entry timer, if any.`,
		Example: `  # Check running timer
  clickup task time running`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTimeRunning(f, &jsonFlags)
		},
	}

	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}

func runTimeRunning(f *cmdutil.Factory, jsonFlags *cmdutil.JSONFlags) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

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

	ctx := context.Background()
	resp, err := apiv2.Getrunningtimeentry(ctx, client, teamID)
	if err != nil {
		return fmt.Errorf("failed to get running timer: %w", err)
	}

	if jsonFlags.WantsJSON() {
		return jsonFlags.OutputJSON(ios.Out, resp)
	}

	data := resp.Data
	if data.ID == "" {
		fmt.Fprintln(ios.Out, "No timer is currently running.")
		return nil
	}

	// Calculate elapsed time from start timestamp.
	elapsed := ""
	if data.Start != "" {
		if startMs, err := strconv.ParseInt(data.Start, 10, 64); err == nil {
			dur := time.Since(time.UnixMilli(startMs))
			hours := int(dur.Hours())
			mins := int(dur.Minutes()) % 60
			if hours > 0 {
				elapsed = fmt.Sprintf("%dh %dm", hours, mins)
			} else {
				elapsed = fmt.Sprintf("%dm", mins)
			}
		}
	}

	fmt.Fprintf(ios.Out, "%s Running timer %s", cs.Green("!"), cs.Gray("("+data.ID+")"))
	if elapsed != "" {
		fmt.Fprintf(ios.Out, " — %s elapsed", cs.Bold(elapsed))
	}
	fmt.Fprintln(ios.Out)

	if data.Task.ID != "" {
		fmt.Fprintf(ios.Out, "  Task: %s %s\n", cs.Bold(data.Task.Name), cs.Gray("#"+data.Task.ID))
	}
	if data.Description != "" {
		fmt.Fprintf(ios.Out, "  Description: %s\n", data.Description)
	}

	return nil
}
