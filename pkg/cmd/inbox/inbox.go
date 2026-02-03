package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type inboxOptions struct {
	factory   *cmdutil.Factory
	days      int
	limit     int
	jsonFlags cmdutil.JSONFlags
}

type mention struct {
	TaskID      string   `json:"task_id"`
	TaskName    string   `json:"task_name"`
	CommentID   string   `json:"comment_id"`
	CommentText string   `json:"comment_text"`
	Attachments []string `json:"attachments,omitempty"`
	Author      string   `json:"author"`
	Date        string   `json:"date"`
	DateMs      int64    `json:"-"`
}

type userResponse struct {
	User struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

type commentResponse struct {
	Comments []commentData `json:"comments"`
}

type commentBlock struct {
	Type  string              `json:"type,omitempty"`
	Text  string              `json:"text,omitempty"`
	Image *commentMediaObject `json:"image,omitempty"`
	Frame *commentMediaObject `json:"frame,omitempty"`
}

type commentMediaObject struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type commentData struct {
	ID          string         `json:"id"`
	CommentText string         `json:"comment_text"`
	Comment     []commentBlock `json:"comment"`
	User        struct {
		Username string `json:"username"`
	} `json:"user"`
	Date string `json:"date"`
}

// NewCmdInbox returns the "inbox" command.
func NewCmdInbox(f *cmdutil.Factory) *cobra.Command {
	opts := &inboxOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Show recent @mentions",
		Long: `Show recent comments that @mention you across your workspace.

Scans recently updated tasks for comments containing your username.
Since ClickUp does not provide a public inbox API, this command
approximates your inbox by searching task comments.`,
		Example: `  # Show mentions from the last 7 days
  clickup inbox

  # Look back 30 days
  clickup inbox --days 30

  # JSON output for scripting
  clickup inbox --json`,
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return inboxRun(opts)
		},
	}

	cmd.Flags().IntVar(&opts.days, "days", 7, "How many days back to search")
	cmd.Flags().IntVar(&opts.limit, "limit", 50, "Maximum number of tasks to scan for mentions")
	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func inboxRun(opts *inboxOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	// Get current user info.
	fmt.Fprintln(ios.ErrOut, "Fetching your user info...")
	user, err := getCurrentUser(client)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Get workspace ID.
	cfg, err := opts.factory.Config()
	if err != nil {
		return err
	}
	teamID := cfg.Workspace
	if teamID == "" {
		return fmt.Errorf("workspace ID required. Set with 'clickup auth login' or 'clickup config set workspace <id>'")
	}

	// Fetch recently updated tasks.
	cutoff := time.Now().AddDate(0, 0, -opts.days)
	cutoffClickup := clickup.NewDate(cutoff)

	fmt.Fprintf(ios.ErrOut, "Scanning tasks updated in the last %s for @%s...\n",
		text.Pluralize(opts.days, "day"), user.User.Username)

	ctx := context.Background()
	taskOpts := &clickup.GetTasksOptions{
		DateUpdatedGt: cutoffClickup,
		IncludeClosed: true,
		Subtasks:      true,
		Page:          0,
	}

	var allTasks []clickup.Task
	for page := 0; len(allTasks) < opts.limit; page++ {
		taskOpts.Page = page
		tasks, _, err := client.Clickup.Tasks.GetFilteredTeamTasks(ctx, teamID, taskOpts)
		if err != nil {
			return fmt.Errorf("failed to fetch tasks: %w", err)
		}
		if len(tasks) == 0 {
			break
		}
		allTasks = append(allTasks, tasks...)
	}

	if len(allTasks) > opts.limit {
		allTasks = allTasks[:opts.limit]
	}

	if len(allTasks) == 0 {
		fmt.Fprintln(ios.ErrOut, "No recently updated tasks found.")
		return nil
	}

	fmt.Fprintf(ios.ErrOut, "Checking comments on %s...\n", text.Pluralize(len(allTasks), "task"))

	// Fetch comments concurrently with a semaphore.
	type taskComments struct {
		task     clickup.Task
		comments []commentData
		err      error
	}

	results := make([]taskComments, len(allTasks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // max 5 concurrent requests

	for i, t := range allTasks {
		wg.Add(1)
		go func(idx int, task clickup.Task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			comments, err := fetchTaskComments(client, task.ID)
			results[idx] = taskComments{
				task:     task,
				comments: comments,
				err:      err,
			}
		}(i, t)
	}
	wg.Wait()

	// Filter for mentions.
	username := strings.ToLower(user.User.Username)
	var mentions []mention

	for _, r := range results {
		if r.err != nil {
			continue
		}
		for _, c := range r.comments {
			if containsMention(c.CommentText, username) && strings.ToLower(c.User.Username) != username {
				ms, _ := strconv.ParseInt(c.Date, 10, 64)
				attachments := extractAttachmentURLs(c.Comment)
				mentions = append(mentions, mention{
					TaskID:      r.task.ID,
					TaskName:    r.task.Name,
					CommentID:   c.ID,
					CommentText: c.CommentText,
					Attachments: attachments,
					Author:      c.User.Username,
					Date:        c.Date,
					DateMs:      ms,
				})
			}
		}
	}

	// Sort by date, oldest first (newest at bottom, closest to cursor).
	sort.Slice(mentions, func(i, j int) bool {
		return mentions[i].DateMs < mentions[j].DateMs
	})

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, mentions)
	}

	if len(mentions) == 0 {
		fmt.Fprintf(ios.Out, "No @mentions found in the last %s.\n", text.Pluralize(opts.days, "day"))
		return nil
	}

	fmt.Fprintf(ios.ErrOut, "\nShowing %s from the last %s\n\n",
		text.Pluralize(len(mentions), "mention"),
		text.Pluralize(opts.days, "day"))

	for i, m := range mentions {
		// Header line: author + task ID + time
		fmt.Fprintf(ios.Out, "%s commented on %s  %s\n",
			cs.Bold(m.Author),
			cs.Cyan("#"+m.TaskID),
			cs.Gray(formatMentionDate(m.Date)),
		)
		// Task name
		fmt.Fprintf(ios.Out, "%s\n", m.TaskName)
		// Comment body, indented
		fmt.Fprintf(ios.Out, "\n  %s\n", strings.TrimSpace(m.CommentText))
		// Attachment URLs
		for _, url := range m.Attachments {
			fmt.Fprintf(ios.Out, "  %s %s\n", cs.Gray("attachment:"), url)
		}
		// Separator between entries
		if i < len(mentions)-1 {
			fmt.Fprintln(ios.Out)
		}
	}

	// Quick actions footer
	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup comment add <task-id> \"@user text\" (supports @mentions)\n", cs.Gray("Reply:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task view <task-id>\n", cs.Gray("View:"))
	fmt.Fprintf(ios.Out, "  %s  clickup inbox --json\n", cs.Gray("JSON:"))

	return nil
}

func getCurrentUser(client *api.Client) (*userResponse, error) {
	req, err := http.NewRequest("GET", "https://api.clickup.com/api/v2/user", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result userResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func fetchTaskComments(client *api.Client, taskID string) ([]commentData, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/comment", taskID)
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result commentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Comments, nil
}

func extractAttachmentURLs(blocks []commentBlock) []string {
	var urls []string
	for _, b := range blocks {
		switch b.Type {
		case "image":
			if b.Image != nil && b.Image.URL != "" {
				urls = append(urls, b.Image.URL)
			}
		case "frame":
			if b.Frame != nil && b.Frame.URL != "" {
				urls = append(urls, b.Frame.URL)
			}
		}
	}
	return urls
}

func containsMention(commentText, username string) bool {
	lower := strings.ToLower(commentText)
	return strings.Contains(lower, "@"+username) ||
		strings.Contains(lower, "@ "+username)
}

func formatMentionDate(dateStr string) string {
	ms, err := strconv.ParseInt(dateStr, 10, 64)
	if err != nil {
		return dateStr
	}
	return text.RelativeTime(time.UnixMilli(ms))
}
