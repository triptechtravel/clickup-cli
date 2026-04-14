package task

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	clickupv3 "github.com/triptechtravel/clickup-cli/api/clickupv3"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type moveOptions struct {
	taskID          string
	listID          string
	moveCustomFields bool
	jsonFlags       cmdutil.JSONFlags
}

// NewCmdMove returns a command to move a task to a different list.
func NewCmdMove(f *cmdutil.Factory) *cobra.Command {
	opts := &moveOptions{}

	cmd := &cobra.Command{
		Use:   "move <task-id>",
		Short: "Move a task to a different list",
		Long: `Move a ClickUp task to a different list.

The task's home list is changed to the target list. Use --move-custom-fields
to carry custom field values from the current list to the new list.`,
		Example: `  # Move a task to a different list
  clickup task move 86abc123 --list 901613544162

  # Move and carry custom fields
  clickup task move 86abc123 --list 901613544162 --move-custom-fields

  # Auto-detect task from branch
  clickup task move --list 901613544162`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.taskID = args[0]
			}
			return runMove(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.listID, "list", "", "Target list ID (required)")
	cmd.Flags().BoolVar(&opts.moveCustomFields, "move-custom-fields", false, "Carry custom fields to the new list")
	_ = cmd.MarkFlagRequired("list")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runMove(f *cmdutil.Factory, opts *moveOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	workspaceID := cfg.Workspace
	if workspaceID == "" {
		return fmt.Errorf("no workspace configured. Run 'clickup auth login' first")
	}

	// Resolve task ID from argument or git branch.
	taskID := opts.taskID
	if taskID == "" {
		gitCtx, gitErr := f.GitContext()
		if gitErr != nil {
			return fmt.Errorf("no task ID provided and could not detect git branch: %w", gitErr)
		}
		if gitCtx.TaskID == nil {
			return fmt.Errorf("no task ID provided and could not detect task from branch %q", gitCtx.Branch)
		}
		taskID = gitCtx.TaskID.ID
	} else {
		parsed := git.ParseTaskID(taskID)
		taskID = parsed.ID
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	req := &clickupv3.TaskMoveTaskBodyParamsDto{}
	if opts.moveCustomFields {
		t := true
		req.MoveCustomFields = &t
	}

	err = apiv3.MoveTask(ctx, client, workspaceID, taskID, opts.listID, req)
	if err != nil {
		return fmt.Errorf("failed to move task %s to list %s: %w", taskID, opts.listID, err)
	}

	// Fetch the updated task for output.
	qs := cmdutil.CustomIDTaskQuery(cfg, false)
	task, fetchErr := apiv2.GetTaskLocal(ctx, client, taskID, qs)

	if opts.jsonFlags.WantsJSON() {
		if fetchErr == nil {
			return opts.jsonFlags.OutputJSON(ios.Out, task)
		}
		// Minimal JSON if fetch failed.
		return opts.jsonFlags.OutputJSON(ios.Out, map[string]string{
			"task_id": taskID,
			"list_id": opts.listID,
			"status":  "moved",
		})
	}

	if fetchErr == nil {
		id := task.ID
		if task.CustomID != "" {
			id = task.CustomID
		}
		fmt.Fprintf(ios.Out, "%s Moved task %s to list %s\n", cs.Green("!"), cs.Bold(task.Name+" #"+id), cs.Cyan(opts.listID))
	} else {
		fmt.Fprintf(ios.Out, "%s Moved task %s to list %s\n", cs.Green("!"), cs.Bold(taskID), cs.Cyan(opts.listID))
	}

	return nil
}
