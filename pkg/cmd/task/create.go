package task

import (
	"context"
	"fmt"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type createOptions struct {
	listID      string
	name        string
	description string
	status      string
	priority    int
	assignees   []int
}

// NewCmdCreate returns a command to create a new ClickUp task.
func NewCmdCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new ClickUp task",
		Long: `Create a new task in a ClickUp list.

The --list-id flag is required to specify which list to create the task in.
If --name is not provided, the command enters interactive mode and prompts
for the task name, description, status, and priority.`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.listID == "" {
				return fmt.Errorf("required flag --list-id not set")
			}
			return runCreate(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.listID, "list-id", "", "ClickUp list ID (required)")
	cmd.Flags().StringVar(&opts.name, "name", "", "Task name")
	cmd.Flags().StringVar(&opts.description, "description", "", "Task description")
	cmd.Flags().StringVar(&opts.status, "status", "", "Task status")
	cmd.Flags().IntVar(&opts.priority, "priority", 0, "Task priority (1=Urgent, 2=High, 3=Normal, 4=Low)")
	cmd.Flags().IntSliceVar(&opts.assignees, "assignee", nil, "Assignee user ID(s)")

	_ = cmd.MarkFlagRequired("list-id")

	return cmd
}

func runCreate(f *cmdutil.Factory, opts *createOptions) error {
	ios := f.IOStreams

	// If name is not provided, enter interactive mode.
	if opts.name == "" {
		if !ios.IsTerminal() {
			return fmt.Errorf("--name is required in non-interactive mode")
		}

		p := prompter.New(ios)

		name, err := p.Input("Task name:", "")
		if err != nil {
			return err
		}
		if name == "" {
			return fmt.Errorf("task name cannot be empty")
		}
		opts.name = name

		description, err := p.Editor("Task description:", "", "*.md")
		if err != nil {
			return err
		}
		opts.description = description

		status, err := p.Input("Status (leave empty for default):", "")
		if err != nil {
			return err
		}
		opts.status = status

		priorityOptions := []string{"None", "Urgent (1)", "High (2)", "Normal (3)", "Low (4)"}
		priorityIdx, err := p.Select("Priority:", priorityOptions)
		if err != nil {
			return err
		}
		// Map selection index to ClickUp priority value (0 = none).
		priorityMap := []int{0, 1, 2, 3, 4}
		opts.priority = priorityMap[priorityIdx]
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	taskReq := &clickup.TaskRequest{
		Name:        opts.name,
		Description: opts.description,
	}

	if opts.status != "" {
		taskReq.Status = opts.status
	}

	if opts.priority > 0 {
		taskReq.Priority = opts.priority
	}

	if len(opts.assignees) > 0 {
		taskReq.Assignees = opts.assignees
	}

	ctx := context.Background()
	task, _, err := client.Clickup.Tasks.CreateTask(ctx, opts.listID, taskReq)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	cs := ios.ColorScheme()
	id := task.ID
	if task.CustomID != "" {
		id = task.CustomID
	}

	fmt.Fprintf(ios.Out, "%s Created task %s %s\n", cs.Green("!"), cs.Bold(task.Name), cs.Gray("#"+id))
	if task.URL != "" {
		fmt.Fprintf(ios.Out, "%s\n", cs.Cyan(task.URL))
	}

	return nil
}
