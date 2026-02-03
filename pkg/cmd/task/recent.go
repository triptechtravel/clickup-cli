package task

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type recentOptions struct {
	factory   *cmdutil.Factory
	limit     int
	all       bool
	sprint    bool
	jsonFlags cmdutil.JSONFlags
}

// NewCmdRecent returns a command to show recently updated tasks.
func NewCmdRecent(f *cmdutil.Factory) *cobra.Command {
	opts := &recentOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "recent",
		Short: "Show recently updated tasks",
		Long: `Show tasks recently updated in your workspace, ordered by last activity.

By default shows your own tasks (assigned to you). Use --all to see all
recently updated tasks across the team.

Each task includes its list and folder location, making it easy to discover
where work is happening when you're unsure which list or folder to search.`,
		Example: `  # Show your recent tasks
  clickup task recent

  # Show all team activity
  clickup task recent --all

  # Only show tasks from the sprint folder
  clickup task recent --sprint

  # Show more results
  clickup task recent --limit 30

  # JSON output for scripting
  clickup task recent --json`,
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRecent(opts)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum number of tasks to show")
	cmd.Flags().BoolVar(&opts.all, "all", false, "Show all team tasks, not just yours")
	cmd.Flags().BoolVar(&opts.sprint, "sprint", false, "Only show tasks from the current sprint folder")
	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runRecent(opts *recentOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	var tasks []cmdutil.RecentTask
	var err error

	if opts.all {
		fmt.Fprintln(ios.ErrOut, "Fetching recently updated tasks...")
		tasks, err = cmdutil.FetchRecentTeamTasks(opts.factory, opts.limit)
	} else {
		fmt.Fprintln(ios.ErrOut, "Fetching your recently updated tasks...")
		tasks, err = cmdutil.FetchRecentTasks(opts.factory, opts.limit)
	}
	if err != nil {
		return err
	}

	// --sprint: filter to tasks in sprint folders only.
	if opts.sprint {
		var sprintTasks []cmdutil.RecentTask
		for _, t := range tasks {
			lower := strings.ToLower(t.FolderName)
			if strings.Contains(lower, "sprint") && !strings.Contains(lower, "archive") {
				sprintTasks = append(sprintTasks, t)
			}
		}
		tasks = sprintTasks
	}

	if len(tasks) == 0 {
		fmt.Fprintln(ios.ErrOut, "No recently updated tasks found.")
		return nil
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, tasks)
	}

	// Show location summary hint.
	locations := cmdutil.LocationSummary(tasks)
	if len(locations) > 0 {
		fmt.Fprintf(ios.ErrOut, "Active locations: %s\n\n", strings.Join(locations, ", "))
	}

	tp := tableprinter.New(ios)
	tp.AddField(cs.Bold("ID"))
	tp.AddField(cs.Bold("NAME"))
	tp.AddField(cs.Bold("STATUS"))
	tp.AddField(cs.Bold("FOLDER"))
	tp.AddField(cs.Bold("LIST"))
	tp.EndRow()
	tp.SetTruncateColumn(1)

	for _, t := range tasks {
		tp.AddField(t.ID)
		tp.AddField(t.Name)
		statusFn := cs.StatusColor(strings.ToLower(t.Status))
		tp.AddField(statusFn(t.Status))
		tp.AddField(t.FolderName)
		tp.AddField(t.ListName)
		tp.EndRow()
	}

	if err := tp.Render(); err != nil {
		return err
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task view <id>\n", cs.Gray("View:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task search <query>\n", cs.Gray("Search:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task edit <id> --status <status>\n", cs.Gray("Edit:"))
	fmt.Fprintf(ios.Out, "  %s  clickup sprint current\n", cs.Gray("Sprint:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task recent --json\n", cs.Gray("JSON:"))

	return nil
}
