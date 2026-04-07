package attachment

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/apiv3"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type listOptions struct {
	factory   *cmdutil.Factory
	taskID    string
	jsonFlags cmdutil.JSONFlags
}

// NewCmdList returns the "attachment list" command.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "list [TASK]",
		Short: "List attachments on a task",
		Long: `List attachments on a ClickUp task.

If TASK is not provided, the task ID is auto-detected from the current git branch.`,
		Args:              cobra.MaximumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 1 {
				opts.taskID = args[0]
			}
			return listRun(opts)
		},
	}

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func listRun(opts *listOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	taskID := opts.taskID
	if taskID == "" {
		gitCtx, err := opts.factory.GitContext()
		if err != nil {
			return fmt.Errorf("could not detect git context: %w\n\n%s", err,
				"Tip: provide the task ID as the first argument")
		}
		if gitCtx.TaskID == nil {
			return fmt.Errorf("%s", git.BranchNamingSuggestion(gitCtx.Branch))
		}
		taskID = gitCtx.TaskID.ID
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Bold(taskID), cs.Cyan(gitCtx.Branch))
	} else {
		parsed := git.ParseTaskID(taskID)
		taskID = parsed.ID
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, err := apiv3.ListAttachments(ctx, client, cfg.Workspace, "attachments", taskID)
	if err != nil {
		return fmt.Errorf("failed to list attachments: %w", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, result.Data)
	}

	if len(result.Data) == 0 {
		fmt.Fprintf(ios.ErrOut, "No attachments found on task %s\n", taskID)
		return nil
	}

	tp := tableprinter.New(ios)
	tp.SetTruncateColumn(4)

	for _, a := range result.Data {
		tp.AddField(a.Title)
		tp.AddField(a.Extension)
		tp.AddField(text.FormatBytes(a.Size))
		tp.AddField(text.RelativeTime(time.UnixMilli(int64(a.DateCreated))))
		tp.AddField(a.URL)
		tp.EndRow()
	}

	if err := tp.Render(); err != nil {
		return err
	}

	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup attachment add %s <file>\n", cs.Gray("Upload:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup attachment list %s --json\n", cs.Gray("JSON:"), taskID)

	return nil
}
