package attachment

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type addOptions struct {
	factory *cmdutil.Factory
	taskID  string
	files   []string
}

// NewCmdAdd returns the "attachment add" command.
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	opts := &addOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "add [TASK] <FILE>...",
		Short: "Upload file(s) to a task",
		Long: `Upload one or more files as attachments to a ClickUp task.

If TASK is not provided, the task ID is auto-detected from the current git branch.
When the first argument is a file that exists on disk, all arguments are treated
as files and the task is auto-detected.`,
		Example: `  # Upload a file to the task detected from the current branch
  clickup attachment add screenshot.png

  # Upload to a specific task
  clickup attachment add abc123 report.pdf

  # Upload multiple files
  clickup attachment add abc123 file1.png file2.pdf`,
		Args:              cobra.MinimumNArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If the first arg is a file on disk, treat all args as files.
			if _, err := os.Stat(args[0]); err == nil {
				opts.files = args
			} else if len(args) < 2 {
				return fmt.Errorf("at least one file is required")
			} else {
				opts.taskID = args[0]
				opts.files = args[1:]
			}
			return addRun(opts)
		},
	}

	return cmd
}

func addRun(opts *addOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	taskID := opts.taskID
	isCustomID := false
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
		isCustomID = gitCtx.TaskID.IsCustomID
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Bold(taskID), cs.Cyan(gitCtx.Branch))
	} else {
		parsed := git.ParseTaskID(taskID)
		taskID = parsed.ID
		isCustomID = parsed.IsCustomID
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return err
	}

	// Build custom task ID options for the v2 API.
	var attachOpts *clickup.TaskAttachementOptions
	if isCustomID {
		attachOpts = &clickup.TaskAttachementOptions{
			CustomTaskIDs: true,
		}
		if cfg.Workspace != "" {
			if tid, err := strconv.Atoi(cfg.Workspace); err == nil {
				attachOpts.TeamID = tid
			}
		}
	}

	ctx := context.Background()

	for _, filePath := range opts.files {
		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", filePath, err)
		}

		info, err := f.Stat()
		if err != nil {
			f.Close()
			return fmt.Errorf("failed to stat %s: %w", filePath, err)
		}

		attachment := &clickup.Attachment{
			FileName: filepath.Base(filePath),
			Reader:   f,
		}

		resp, _, err := client.Clickup.Attachments.CreateTaskAttachment(ctx, taskID, attachOpts, attachment)
		f.Close()
		if err != nil {
			return fmt.Errorf("failed to upload %s: %w", filepath.Base(filePath), err)
		}

		fmt.Fprintf(ios.Out, "%s Uploaded %s (%s) to task %s\n",
			cs.Green("!"),
			cs.Bold(filepath.Base(filePath)),
			text.FormatBytes(int(info.Size())),
			cs.Bold(taskID),
		)
		if resp.URL != "" {
			fmt.Fprintf(ios.Out, "  %s %s\n", cs.Gray("URL:"), resp.URL)
		}
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup attachment list %s\n", cs.Gray("List:"), taskID)
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), taskID)

	return nil
}
