package status

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type setOptions struct {
	factory      *cmdutil.Factory
	targetStatus string
	taskID       string
}

// NewCmdSet returns the "status set" command.
func NewCmdSet(f *cmdutil.Factory) *cobra.Command {
	opts := &setOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "set <status> [task]",
		Short: "Set the status of a task",
		Long: `Change a task's status using fuzzy matching.

The STATUS argument is matched against available statuses for the task's list (or space if the list has no custom statuses).
Matching priority: exact match, then case-insensitive contains, then fuzzy match.

If TASK is not provided, the task ID is auto-detected from the current git branch.`,
		Example: `  # Set status using auto-detected task from branch
  clickup status set "in progress"

  # Set status for a specific task
  clickup status set "done" CU-abc123

  # Fuzzy matching works too
  clickup status set "prog" CU-abc123`,
		Args:               cobra.RangeArgs(1, 2),
		PersistentPreRunE:  cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetStatus = args[0]
			if len(args) > 1 {
				opts.taskID = args[1]
			}
			return setRun(opts)
		},
	}

	return cmd
}

func setRun(opts *setOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID.
	taskID := opts.taskID
	isCustomID := false
	if taskID == "" {
		gitCtx, err := opts.factory.GitContext()
		if err != nil {
			return fmt.Errorf("failed to detect git context: %w\n\n%s", err, git.BranchNamingSuggestion(""))
		}
		if gitCtx.TaskID == nil {
			return fmt.Errorf("no task ID provided and none detected from branch\n\n%s", git.BranchNamingSuggestion(gitCtx.Branch))
		}
		taskID = gitCtx.TaskID.ID
		isCustomID = gitCtx.TaskID.IsCustomID
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Cyan(taskID), cs.Cyan(gitCtx.Branch))
	} else {
		parsed := git.ParseTaskID(taskID)
		taskID = parsed.ID
		isCustomID = parsed.IsCustomID
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return err
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Fetch the task to determine its space.
	qs := cmdutil.CustomIDTaskQuery(cfg, isCustomID)
	task, err := apiv2.GetTaskLocal(ctx, client, taskID, qs)
	if err != nil {
		return fmt.Errorf("failed to get task %s: %w", taskID, err)
	}

	currentStatus := task.Status.Status

	// Validate status against the task's list (with space fallback).
	matched, err := cmdutil.ValidateStatusWithList(client, task.Space.ID, task.List.ID, opts.targetStatus, ios.ErrOut)
	if err != nil {
		return err
	}

	// Update the task status.
	payload := map[string]string{"status": matched}
	if err := apiv2.Do(ctx, client, "PUT", fmt.Sprintf("task/%s", task.ID), payload, nil); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Report success.
	fromColor := cs.StatusColor(strings.ToLower(currentStatus))
	toColor := cs.StatusColor(strings.ToLower(matched))
	fmt.Fprintf(ios.Out, "Status changed: %s %s %s\n",
		fromColor(fmt.Sprintf("'%s'", currentStatus)),
		cs.Gray("\u2192"),
		toColor(fmt.Sprintf("'%s'", matched)),
	)

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup comment add %s \"@user text\" (supports @mentions)\n", cs.Gray("Comment:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s --json\n", cs.Gray("JSON:"), taskID)

	return nil
}

