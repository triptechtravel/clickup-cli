package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type listOptions struct {
	listID    string
	assignee  []string
	status    []string
	sprint    string
	page      int
	jsonFlags cmdutil.JSONFlags
}

// NewCmdList returns a command to list ClickUp tasks in a given list.
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks in a ClickUp list",
		Long: `List tasks from a ClickUp list with optional filters.

The --list-id flag is required to specify which ClickUp list to query.
Results can be filtered by assignee, status, and sprint.`,
		Example: `  # List tasks in a ClickUp list
  clickup task list --list-id 12345

  # Filter by assignee and status
  clickup task list --list-id 12345 --assignee me --status "in progress"`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.listID == "" {
				return fmt.Errorf("required flag --list-id not set")
			}
			return runList(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.listID, "list-id", "", "ClickUp list ID (required)")
	cmd.Flags().StringSliceVar(&opts.assignee, "assignee", nil, `Filter by assignee ID(s), or "me" for yourself`)
	cmd.Flags().StringSliceVar(&opts.status, "status", nil, "Filter by status(es)")
	cmd.Flags().StringVar(&opts.sprint, "sprint", "", "Filter by sprint name")
	cmd.Flags().IntVar(&opts.page, "page", 0, "Page number for pagination (starts at 0)")

	_ = cmd.MarkFlagRequired("list-id")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runList(f *cmdutil.Factory, opts *listOptions) error {
	ios := f.IOStreams

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	taskOpts := &clickup.GetTasksOptions{
		Page: opts.page,
	}

	if len(opts.status) > 0 {
		taskOpts.Statuses = opts.status
	}

	if len(opts.assignee) > 0 {
		taskOpts.Assignees = opts.assignee
	}

	if opts.sprint != "" {
		// Sprint filtering is handled via tags in ClickUp.
		taskOpts.Tags = []string{opts.sprint}
	}

	ctx := context.Background()
	tasks, _, err := client.Clickup.Tasks.GetTasks(ctx, opts.listID, taskOpts)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Fprintln(ios.ErrOut, "No tasks found.")
		return nil
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, tasks)
	}

	return printTaskTable(f, tasks)
}

func printTaskTable(f *cmdutil.Factory, tasks []clickup.Task) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	tp := tableprinter.New(ios)

	// Header row
	tp.AddField(cs.Bold("ID"))
	tp.AddField(cs.Bold("NAME"))
	tp.AddField(cs.Bold("STATUS"))
	tp.AddField(cs.Bold("PRIORITY"))
	tp.AddField(cs.Bold("ASSIGNEE"))
	tp.AddField(cs.Bold("TAGS"))
	tp.AddField(cs.Bold("DUE"))
	tp.EndRow()

	tp.SetTruncateColumn(1) // Truncate name column if table is too wide.

	for _, t := range tasks {
		id := t.ID
		if t.CustomID != "" {
			id = t.CustomID
		}
		tp.AddField(id)
		tp.AddField(t.Name)

		statusText := t.Status.Status
		statusColorFn := cs.StatusColor(strings.ToLower(statusText))
		tp.AddField(statusColorFn(statusText))

		tp.AddField(t.Priority.Priority)

		assigneeNames := make([]string, 0, len(t.Assignees))
		for _, a := range t.Assignees {
			assigneeNames = append(assigneeNames, a.Username)
		}
		tp.AddField(strings.Join(assigneeNames, ", "))

		tagNames := make([]string, 0, len(t.Tags))
		for _, tag := range t.Tags {
			tagNames = append(tagNames, tag.Name)
		}
		tp.AddField(strings.Join(tagNames, ", "))

		var dueStr string
		if t.DueDate != nil {
			if dt := t.DueDate.Time(); dt != nil {
				dueStr = dt.Format("Jan 02")
			}
		}
		tp.AddField(dueStr)

		tp.EndRow()
	}

	return tp.Render()
}
