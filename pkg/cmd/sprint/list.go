package sprint

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// NewCmdSprintList returns the sprint list command.
func NewCmdSprintList(f *cmdutil.Factory) *cobra.Command {
	var jsonFlags cmdutil.JSONFlags
	var folderID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sprints in a folder",
		Long: `List sprints (lists) within a sprint folder.

Sprints in ClickUp are organized as lists within a folder.
This command finds sprint folders in your configured space
and lists the sprints within them.

If multiple sprint folders exist, use --folder to specify one,
or the CLI will remember your choice.`,
		Example: `  # List all sprints
  clickup sprint list

  # Specify sprint folder
  clickup sprint list --folder 132693664`,
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSprintList(f, folderID, &jsonFlags)
		},
	}

	cmd.Flags().StringVar(&folderID, "folder", "", "Sprint folder ID (auto-detected if not set)")
	cmdutil.AddJSONFlags(cmd, &jsonFlags)

	return cmd
}

type sprintListEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TaskCount string `json:"task_count"`
	StartDate string `json:"start_date"`
	DueDate   string `json:"due_date"`
	Status    string `json:"status"`
}

func runSprintList(f *cmdutil.Factory, folderID string, jsonFlags *cmdutil.JSONFlags) error {
	ios := f.IOStreams
	cs := ios.ColorScheme()
	ctx := context.Background()

	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	cfg, err := f.Config()
	if err != nil {
		return err
	}

	// Resolve the sprint folder.
	if folderID != "" {
		// Explicitly provided — save for next time.
		if cfg.SprintFolder != folderID {
			cfg.SprintFolder = folderID
			_ = cfg.Save()
		}
	} else {
		folderID = cfg.SprintFolder
	}
	if folderID == "" {
		// Auto-discover sprint folders.
		spaceID := cfg.Space
		if spaceID == "" {
			return fmt.Errorf("no space configured. Run 'clickup space select' first")
		}

		folders, _, err := client.Clickup.Folders.GetFolders(ctx, spaceID, false)
		if err != nil {
			return fmt.Errorf("failed to list folders: %w", err)
		}

		var sprintFolders []clickup.Folder
		for _, folder := range folders {
			if strings.Contains(strings.ToLower(folder.Name), "sprint") &&
				!strings.Contains(strings.ToLower(folder.Name), "archive") {
				sprintFolders = append(sprintFolders, folder)
			}
		}

		if len(sprintFolders) == 0 {
			return fmt.Errorf("no sprint folders found in space. Use --folder to specify a folder ID")
		}

		if len(sprintFolders) == 1 {
			folderID = sprintFolders[0].ID
			// Save for next time.
			cfg.SprintFolder = folderID
			_ = cfg.Save()
			fmt.Fprintf(ios.ErrOut, "Using sprint folder: %s\n", sprintFolders[0].Name)
		} else {
			fmt.Fprintln(ios.ErrOut, "Multiple sprint folders found:")
			for _, sf := range sprintFolders {
				fmt.Fprintf(ios.ErrOut, "  %s  %s\n", sf.ID, sf.Name)
			}
			return fmt.Errorf("use --folder <id> to select one")
		}
	}

	// Get lists (sprints) in the folder.
	lists, _, err := client.Clickup.Lists.GetLists(ctx, folderID, false)
	if err != nil {
		return fmt.Errorf("failed to list sprints: %w", err)
	}

	if len(lists) == 0 {
		fmt.Fprintln(ios.Out, "No sprints found in this folder.")
		return nil
	}

	// Build entries with status based on dates.
	now := time.Now()
	entries := make([]sprintListEntry, 0, len(lists))

	for _, l := range lists {
		tc, _ := l.TaskCount.Int64()
		start := parseMSTimestamp(l.StartDate)
		due := parseMSTimestamp(l.DueDate)

		status := classifySprint(start, due, now)

		entries = append(entries, sprintListEntry{
			ID:        l.ID,
			Name:      l.Name,
			TaskCount: strconv.FormatInt(tc, 10),
			StartDate: l.StartDate,
			DueDate:   l.DueDate,
			Status:    status,
		})
	}

	if jsonFlags.WantsJSON() {
		return jsonFlags.OutputJSON(ios.Out, entries)
	}

	tp := tableprinter.New(ios)

	tp.AddField(cs.Bold("ID"))
	tp.AddField(cs.Bold("SPRINT"))
	tp.AddField(cs.Bold("STATUS"))
	tp.AddField(cs.Bold("DATES"))
	tp.AddField(cs.Bold("TASKS"))
	tp.EndRow()

	for _, e := range entries {
		tp.AddField(e.ID)
		tp.AddField(e.Name)

		statusFn := cs.StatusColor(strings.ToLower(e.Status))
		tp.AddField(statusFn(e.Status))

		start := parseMSTimestamp(e.StartDate)
		due := parseMSTimestamp(e.DueDate)
		tp.AddField(formatDateRange(start, due))

		tp.AddField(e.TaskCount)
		tp.EndRow()
	}

	return tp.Render()
}

func classifySprint(start, due, now time.Time) string {
	if start.IsZero() && due.IsZero() {
		return "unknown"
	}
	if !due.IsZero() && now.After(due) {
		return "complete"
	}
	if !start.IsZero() && !due.IsZero() && !now.Before(start) && !now.After(due) {
		return "in progress"
	}
	if !start.IsZero() && now.Before(start) {
		return "upcoming"
	}
	return "unknown"
}

func formatDateRange(start, end time.Time) string {
	if start.IsZero() && end.IsZero() {
		return "-"
	}
	if start.IsZero() {
		return fmt.Sprintf("ends %s", end.Format("Jan 02"))
	}
	if end.IsZero() {
		return fmt.Sprintf("started %s", start.Format("Jan 02"))
	}
	return fmt.Sprintf("%s - %s", start.Format("Jan 02"), end.Format("Jan 02"))
}

func parseMSTimestamp(ms string) time.Time {
	if ms == "" || ms == "0" {
		return time.Time{}
	}
	n, err := strconv.ParseInt(ms, 10, 64)
	if err != nil || n == 0 {
		return time.Time{}
	}
	return time.UnixMilli(n)
}
