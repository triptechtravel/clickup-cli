package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

const pointsNotSet = -999.0

type createOptions struct {
	listID              string
	name                string
	description         string
	markdownDescription string
	status              string
	priority            int
	assignees           []int
	tags                []string
	dueDate             string
	startDate           string
	timeEstimate        string
	points              float64
	parent              string
	linksTo             string
	dueDateTime         bool
	startDateTime       bool
	notifyAll           bool
	customItemID        int
	fields              []string
	jsonFlags           cmdutil.JSONFlags
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
for the task name, description, status, priority, due date, and time estimate.

Additional properties can be set with flags:
  --tags           Tags to add (comma-separated or repeat flag)
  --due-date       Due date in YYYY-MM-DD format
  --start-date     Start date in YYYY-MM-DD format
  --time-estimate  Time estimate (e.g. "2h", "30m", "1h30m")
  --points         Sprint/story points
  --field          Set a custom field ("Name=value", repeatable)
  --parent         Create as subtask of another task
  --type           Task type (0=task, 1=milestone)`,
		Example: `  # Create with flags
  clickup task create --list-id 12345 --name "Fix login bug" --priority 2

  # Interactive mode (prompts for details)
  clickup task create --list-id 12345

  # Create with custom field and due date
  clickup task create --list-id 12345 --name "Deploy v2" --field "Environment=staging" --due-date 2025-03-01

  # Create a subtask
  clickup task create --list-id 12345 --name "Write tests" --parent 86abc123`,
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
	cmd.Flags().StringVar(&opts.markdownDescription, "markdown-description", "", "Task description in markdown")
	cmd.Flags().StringVar(&opts.status, "status", "", "Task status")
	cmd.Flags().IntVar(&opts.priority, "priority", 0, "Task priority (1=Urgent, 2=High, 3=Normal, 4=Low)")
	cmd.Flags().IntSliceVar(&opts.assignees, "assignee", nil, "Assignee user ID(s)")
	cmd.Flags().StringSliceVar(&opts.tags, "tags", nil, "Tags to add to the task")
	cmd.Flags().StringVar(&opts.dueDate, "due-date", "", "Due date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.startDate, "start-date", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.timeEstimate, "time-estimate", "", "Time estimate (e.g. 2h, 30m, 1h30m)")
	cmd.Flags().Float64Var(&opts.points, "points", pointsNotSet, "Sprint/story points")
	cmd.Flags().StringVar(&opts.parent, "parent", "", "Parent task ID (create as subtask)")
	cmd.Flags().StringVar(&opts.linksTo, "links-to", "", "Link to another task by ID")
	cmd.Flags().BoolVar(&opts.dueDateTime, "due-date-time", false, "Include time component in due date")
	cmd.Flags().BoolVar(&opts.startDateTime, "start-date-time", false, "Include time component in start date")
	cmd.Flags().BoolVar(&opts.notifyAll, "notify-all", false, "Notify all assignees and watchers")
	cmd.Flags().IntVar(&opts.customItemID, "type", -1, "Task type (0=task, 1=milestone, or custom type ID)")
	cmd.Flags().StringArrayVar(&opts.fields, "field", nil, `Set a custom field value ("Name=value", repeatable)`)

	_ = cmd.MarkFlagRequired("list-id")

	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

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

		dueDate, err := p.Input("Due date (YYYY-MM-DD, leave empty for none):", "")
		if err != nil {
			return err
		}
		opts.dueDate = dueDate

		timeEstimate, err := p.Input("Time estimate (e.g. 2h, 30m, 1h30m, leave empty for none):", "")
		if err != nil {
			return err
		}
		opts.timeEstimate = timeEstimate
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	taskReq := &clickup.TaskRequest{
		Name:        opts.name,
		Description: opts.description,
	}

	if opts.markdownDescription != "" {
		taskReq.MarkdownDescription = opts.markdownDescription
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

	if len(opts.tags) > 0 {
		taskReq.Tags = opts.tags
	}

	if opts.dueDate != "" {
		d, err := parseDate(opts.dueDate)
		if err != nil {
			return err
		}
		taskReq.DueDate = d
	}

	if opts.startDate != "" {
		d, err := parseDate(opts.startDate)
		if err != nil {
			return err
		}
		taskReq.StartDate = d
	}

	if opts.timeEstimate != "" {
		ms, err := parseDuration(opts.timeEstimate)
		if err != nil {
			return err
		}
		taskReq.TimeEstimate = ms
	}

	if opts.parent != "" {
		taskReq.Parent = opts.parent
	}
	if opts.linksTo != "" {
		taskReq.LinksTo = opts.linksTo
	}
	if opts.dueDateTime {
		taskReq.DueDateTime = true
	}
	if opts.startDateTime {
		taskReq.StartDateTime = true
	}
	if opts.notifyAll {
		taskReq.NotifyAll = true
	}
	if opts.customItemID >= 0 {
		taskReq.CustomItemId = opts.customItemID
	}

	ctx := context.Background()
	task, _, err := client.Clickup.Tasks.CreateTask(ctx, opts.listID, taskReq)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Set points via raw API call if specified (not supported by go-clickup library).
	if opts.points != pointsNotSet {
		if err := setTaskPoints(client, task.ID, opts.points); err != nil {
			return fmt.Errorf("task created but failed to set points: %w", err)
		}
	}

	// Handle custom field set operations.
	if len(opts.fields) > 0 {
		// Fetch accessible custom fields for the list to resolve names to IDs.
		listFields, _, err := client.Clickup.CustomFields.GetAccessibleCustomFields(ctx, opts.listID)
		if err != nil {
			return fmt.Errorf("task created but failed to fetch custom fields: %w", err)
		}
		for _, fieldSpec := range opts.fields {
			parts := strings.SplitN(fieldSpec, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid --field format %q (use \"Name=value\")", fieldSpec)
			}
			fieldName, fieldValue := parts[0], parts[1]

			cf := resolveFieldByName(listFields, fieldName)
			if cf == nil {
				return fmt.Errorf("custom field %q not found (available: %s)", fieldName, customFieldNames(listFields))
			}

			parsed, err := parseFieldValue(cf, fieldValue)
			if err != nil {
				return err
			}

			_, err = client.Clickup.CustomFields.SetCustomFieldValue(ctx, task.ID, cf.ID, map[string]interface{}{"value": parsed}, nil)
			if err != nil {
				return fmt.Errorf("task created but failed to set custom field %q: %w", fieldName, err)
			}
		}
	}

	cs := ios.ColorScheme()
	id := task.ID
	if task.CustomID != "" {
		id = task.CustomID
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, task)
	}

	fmt.Fprintf(ios.Out, "%s Created task %s %s\n", cs.Green("!"), cs.Bold(task.Name), cs.Gray("#"+id))
	if task.URL != "" {
		fmt.Fprintf(ios.Out, "%s\n", cs.Cyan(task.URL))
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task view %s\n", cs.Gray("View:"), id)
	fmt.Fprintf(ios.Out, "  %s  clickup task edit %s --status <status>\n", cs.Gray("Edit:"), id)
	fmt.Fprintf(ios.Out, "  %s  clickup link pr --task %s\n", cs.Gray("Link PR:"), id)
	fmt.Fprintf(ios.Out, "  %s  clickup comment add %s \"@user text\" (supports @mentions)\n", cs.Gray("Comment:"), id)

	return nil
}
