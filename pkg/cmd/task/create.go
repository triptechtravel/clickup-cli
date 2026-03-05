package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

const pointsNotSet = -999.0

type createOptions struct {
	listID              string
	currentSprint       bool
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
	fromFile            string
	jsonFlags           cmdutil.JSONFlags
}

type taskFileEntry struct {
	Name                string   `json:"name"`
	Description         string   `json:"description,omitempty"`
	MarkdownDescription string   `json:"markdown_description,omitempty"`
	Status              string   `json:"status,omitempty"`
	Priority            int      `json:"priority,omitempty"`
	Assignees           []int    `json:"assignees,omitempty"`
	Tags                []string `json:"tags,omitempty"`
	DueDate             string   `json:"due_date,omitempty"`
	StartDate           string   `json:"start_date,omitempty"`
	TimeEstimate        string   `json:"time_estimate,omitempty"`
	Points              float64  `json:"points,omitempty"`
	Parent              string   `json:"parent,omitempty"`
	LinksTo             string   `json:"links_to,omitempty"`
	Type                int      `json:"type,omitempty"`
	Fields              []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"fields,omitempty"`
}

// NewCmdCreate returns a command to create a new ClickUp task.
func NewCmdCreate(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new ClickUp task",
		Long: `Create a new task in a ClickUp list.

Either --list-id or --current is required. Use --current to automatically
resolve the active sprint list from the configured sprint folder.
If --name is not provided, the command enters interactive mode and prompts
for the task name, description, status, priority, due date, and time estimate.

Use --from-file to bulk create tasks from a JSON file. The file should
contain an array of task objects. Each object supports the same fields as
the CLI flags.

Additional properties can be set with flags:
  --tags           Tags to add (comma-separated or repeat flag)
  --due-date       Due date in YYYY-MM-DD format
  --start-date     Start date in YYYY-MM-DD format
  --time-estimate  Time estimate (e.g. "2h", "30m", "1h30m")
  --points         Sprint/story points
  --field          Set a custom field ("Name=value", repeatable)
  --parent         Create as subtask of another task
  --type           Task type (0=task, 1=milestone)`,
		Example: `  # Create in the current sprint (auto-resolves list from sprint folder)
  clickup task create --current \
    --name "[Bug] Auth — Fix login timeout (API)" --priority 2

  # Create with explicit list ID
  clickup task create --list-id 12345 \
    --name "[Bug] Auth — Fix login timeout (API)" --priority 2

  # Interactive mode (prompts for details)
  clickup task create --current

  # Create with custom field and due date
  clickup task create --current \
    --name "[Feature] Deploy — Release v2 to staging" \
    --field "Environment=staging" --due-date 2025-03-01

  # Create a subtask
  clickup task create --list-id 12345 --name "Write tests" --parent 86abc123

  # Bulk create from JSON file
  clickup task create --current --from-file tasks.json

  # Bulk create subtasks under a parent
  clickup task create --list-id 12345 --from-file checklist.json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.currentSprint {
				listID, err := resolveCurrentSprintList(f)
				if err != nil {
					return err
				}
				opts.listID = listID
			}
			if opts.listID == "" {
				return fmt.Errorf("either --list-id or --current is required")
			}
			if opts.fromFile != "" {
				return runBulkCreate(f, opts)
			}
			return runCreate(f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.listID, "list-id", "", "ClickUp list ID")
	cmd.Flags().BoolVar(&opts.currentSprint, "current", false, "Create in the current sprint (auto-resolves list ID from sprint folder)")
	cmd.Flags().StringVar(&opts.name, "name", "", "Task name (convention: [Type] Context — Action (Platform))")
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
	cmd.Flags().StringVar(&opts.fromFile, "from-file", "", "Create tasks from a JSON file (array of task objects)")

	cmd.MarkFlagsMutuallyExclusive("list-id", "current")

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

	ctx := context.Background()

	taskReq := &clickup.TaskRequest{
		Name:        opts.name,
		Description: opts.description,
	}

	if opts.markdownDescription != "" {
		taskReq.MarkdownDescription = opts.markdownDescription
	}

	// Fetch the list once if we need it for status or tag validation.
	var spaceID string
	if opts.status != "" || len(opts.tags) > 0 {
		list, _, listErr := client.Clickup.Lists.GetList(ctx, opts.listID)
		if listErr == nil && list.Space.ID != "" {
			spaceID = list.Space.ID
		}
	}

	if opts.status != "" {
		if spaceID != "" {
			validated, valErr := cmdutil.ValidateStatus(client, spaceID, opts.status, ios.ErrOut)
			if valErr != nil {
				return valErr
			}
			opts.status = validated
		}
		taskReq.Status = opts.status
	}

	if opts.priority > 0 {
		taskReq.Priority = opts.priority
	}

	if len(opts.assignees) > 0 {
		taskReq.Assignees = opts.assignees
	}

	if len(opts.tags) > 0 {
		if spaceID != "" {
			opts.tags = cmdutil.ValidateTags(client, spaceID, opts.tags, ios.ErrOut)
		}
		// Tags are applied via dedicated API calls after task creation (below),
		// not via the request body which ClickUp ignores.
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

	task, _, err := client.Clickup.Tasks.CreateTask(ctx, opts.listID, taskReq)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Set tags via dedicated API calls (not supported via task request body).
	if len(opts.tags) > 0 {
		if err := addTaskTags(client, task.ID, opts.tags); err != nil {
			return fmt.Errorf("task created but failed to set tags: %w", err)
		}
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

func runBulkCreate(f *cmdutil.Factory, opts *createOptions) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()

	data, err := os.ReadFile(opts.fromFile)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", opts.fromFile, err)
	}

	var entries []taskFileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no tasks found in %s", opts.fromFile)
	}

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Fetch the list once for status/tag validation.
	var spaceID string
	needsValidation := false
	for _, e := range entries {
		if e.Status != "" || len(e.Tags) > 0 {
			needsValidation = true
			break
		}
	}
	if needsValidation {
		list, _, listErr := client.Clickup.Lists.GetList(ctx, opts.listID)
		if listErr == nil && list.Space.ID != "" {
			spaceID = list.Space.ID
		}
	}

	// Fetch custom fields once if any entry uses them.
	var listFields []clickup.CustomField
	needsFields := false
	for _, e := range entries {
		if len(e.Fields) > 0 {
			needsFields = true
			break
		}
	}
	if needsFields {
		fields, _, err := client.Clickup.CustomFields.GetAccessibleCustomFields(ctx, opts.listID)
		if err != nil {
			return fmt.Errorf("failed to fetch custom fields: %w", err)
		}
		listFields = fields
	}

	total := len(entries)
	var created int
	var results []*clickup.Task

	for i, entry := range entries {
		if entry.Name == "" {
			fmt.Fprintf(ios.ErrOut, "%s (%d/%d) Skipped: task name is empty\n", cs.Yellow("!"), i+1, total)
			continue
		}

		taskReq := &clickup.TaskRequest{
			Name:        entry.Name,
			Description: entry.Description,
		}

		if entry.MarkdownDescription != "" {
			taskReq.MarkdownDescription = entry.MarkdownDescription
		}

		if entry.Status != "" {
			status := entry.Status
			if spaceID != "" {
				validated, valErr := cmdutil.ValidateStatus(client, spaceID, status, ios.ErrOut)
				if valErr != nil {
					fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: %v\n", cs.Yellow("!"), i+1, total, entry.Name, valErr)
					continue
				}
				status = validated
			}
			taskReq.Status = status
		}

		if entry.Priority > 0 {
			taskReq.Priority = entry.Priority
		}
		if len(entry.Assignees) > 0 {
			taskReq.Assignees = entry.Assignees
		}

		tags := entry.Tags
		if len(tags) > 0 && spaceID != "" {
			tags = cmdutil.ValidateTags(client, spaceID, tags, ios.ErrOut)
		}

		if entry.DueDate != "" {
			d, err := parseDate(entry.DueDate)
			if err != nil {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: invalid due_date: %v\n", cs.Yellow("!"), i+1, total, entry.Name, err)
				continue
			}
			taskReq.DueDate = d
		}

		if entry.StartDate != "" {
			d, err := parseDate(entry.StartDate)
			if err != nil {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: invalid start_date: %v\n", cs.Yellow("!"), i+1, total, entry.Name, err)
				continue
			}
			taskReq.StartDate = d
		}

		if entry.TimeEstimate != "" {
			ms, err := parseDuration(entry.TimeEstimate)
			if err != nil {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: invalid time_estimate: %v\n", cs.Yellow("!"), i+1, total, entry.Name, err)
				continue
			}
			taskReq.TimeEstimate = ms
		}

		if entry.Parent != "" {
			taskReq.Parent = entry.Parent
		}
		if entry.LinksTo != "" {
			taskReq.LinksTo = entry.LinksTo
		}
		if entry.Type > 0 {
			taskReq.CustomItemId = entry.Type
		}

		task, _, err := client.Clickup.Tasks.CreateTask(ctx, opts.listID, taskReq)
		if err != nil {
			fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: %v\n", cs.Yellow("!"), i+1, total, entry.Name, err)
			continue
		}

		// Set tags via dedicated API calls.
		if len(tags) > 0 {
			if err := addTaskTags(client, task.ID, tags); err != nil {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: created but failed to set tags: %v\n", cs.Yellow("!"), i+1, total, entry.Name, err)
			}
		}

		// Set points via raw API call.
		if entry.Points != 0 {
			if err := setTaskPoints(client, task.ID, entry.Points); err != nil {
				fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: created but failed to set points: %v\n", cs.Yellow("!"), i+1, total, entry.Name, err)
			}
		}

		// Set custom fields.
		if len(entry.Fields) > 0 && listFields != nil {
			for _, fieldSpec := range entry.Fields {
				cf := resolveFieldByName(listFields, fieldSpec.Name)
				if cf == nil {
					fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: custom field %q not found\n", cs.Yellow("!"), i+1, total, entry.Name, fieldSpec.Name)
					continue
				}
				parsed, err := parseFieldValue(cf, fieldSpec.Value)
				if err != nil {
					fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: custom field %q: %v\n", cs.Yellow("!"), i+1, total, entry.Name, fieldSpec.Name, err)
					continue
				}
				_, err = client.Clickup.CustomFields.SetCustomFieldValue(ctx, task.ID, cf.ID, map[string]interface{}{"value": parsed}, nil)
				if err != nil {
					fmt.Fprintf(ios.ErrOut, "%s (%d/%d) %s: failed to set custom field %q: %v\n", cs.Yellow("!"), i+1, total, entry.Name, fieldSpec.Name, err)
				}
			}
		}

		id := task.ID
		if task.CustomID != "" {
			id = task.CustomID
		}

		created++
		results = append(results, task)

		if !opts.jsonFlags.WantsJSON() {
			fmt.Fprintf(ios.Out, "(%d/%d) Created task %s %s\n", i+1, total, cs.Bold(task.Name), cs.Gray("#"+id))
		}
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, results)
	}

	fmt.Fprintf(ios.Out, "\n%s Created %d/%d tasks\n", cs.Green("!"), created, total)
	return nil
}
