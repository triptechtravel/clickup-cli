package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
	"github.com/triptechtravel/clickup-cli/internal/text"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type inboxOptions struct {
	factory   *cmdutil.Factory
	days      int
	limit     int
	noCache   bool
	jsonFlags cmdutil.JSONFlags
}

type mention struct {
	TaskID        string   `json:"task_id"`
	TaskName      string   `json:"task_name"`
	Type          string   `json:"type"`
	CommentID     string   `json:"comment_id,omitempty"`
	CommentText   string   `json:"comment_text,omitempty"`
	Attachments   []string `json:"attachments,omitempty"`
	Author        string   `json:"author"`
	Date          string   `json:"date"`
	DateMs        int64    `json:"-"`
	IsDescription bool     `json:"is_description,omitempty"`
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
		Short: "Show recent @mentions and assignments",
		Long: `Show recent comments that @mention you and tasks newly assigned to you.

Combines two scans: a filtered call for tasks where you are an assignee
(used to detect newly assigned tasks within the lookback window) and a
workspace-wide scan for cold @mentions on tasks you are not assigned to.

Results from the workspace scan are cached locally at
~/.config/clickup/inbox_cache.json by task date_updated, so subsequent
runs only re-fetch comments for tasks that have changed. The cache
expires after 24 hours; pass --no-cache to force a full rescan (the cache
is still rewritten after a --no-cache run so subsequent runs are cheap).

Since ClickUp does not provide a public inbox API, this command
approximates your inbox by combining these two endpoints.`,
		Example: `  # Show mentions and assignments from the last 7 days
  clickup inbox

  # Look back 30 days
  clickup inbox --days 30

  # Force a full rescan, ignoring the cache
  clickup inbox --no-cache

  # JSON output for scripting
  clickup inbox --json`,
		PreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return inboxRun(opts)
		},
	}

	cmd.Flags().IntVar(&opts.days, "days", 7, "How many days back to search")
	cmd.Flags().IntVar(&opts.limit, "limit", 200, "Maximum number of tasks to scan for mentions")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "Bypass the local cache and re-fetch comments for every task")
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

	fmt.Fprintln(ios.ErrOut, "Fetching your user info...")
	user, err := getCurrentUser(client)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return err
	}
	teamID := cfg.Workspace
	if teamID == "" {
		return fmt.Errorf("workspace ID required. Set with 'clickup auth login' or 'clickup config set workspace <id>'")
	}

	cutoff := time.Now().AddDate(0, 0, -opts.days)
	cutoffMs := cutoff.UnixMilli()

	ctx := context.Background()

	// Phase 1: cheap filtered call for tasks where the user is currently an
	// assignee. Used to detect newly assigned tasks within the lookback window.
	fmt.Fprintf(ios.ErrOut, "Fetching tasks assigned to @%s...\n", user.User.Username)
	assignedTasks, err := fetchTeamTasksLocal(ctx, client, teamID, apiv2.FilteredTeamTasksParams{
		DateUpdGt:     cutoffMs,
		Assignees:     []string{strconv.Itoa(user.User.ID)},
		IncludeClosed: true,
		Subtasks:      true,
	}, opts.limit)
	if err != nil {
		return fmt.Errorf("failed to fetch assigned tasks: %w", err)
	}

	// Phase 2: workspace-wide scan for cold mentions. Same call the original
	// inbox made, capped at opts.limit.
	fmt.Fprintf(ios.ErrOut, "Scanning tasks updated in the last %s...\n",
		text.Pluralize(opts.days, "day"))
	workspaceTasks, err := fetchTeamTasksLocal(ctx, client, teamID, apiv2.FilteredTeamTasksParams{
		DateUpdGt:     cutoffMs,
		IncludeClosed: true,
		Subtasks:      true,
	}, opts.limit)
	if err != nil {
		return fmt.Errorf("failed to fetch tasks: %w", err)
	}

	// Phase 3: load cache and diff. A stale or missing cache reverts to a cold
	// scan automatically; --no-cache forces it.
	var cache *inboxCache
	cachePath := inboxCachePath()
	if !opts.noCache {
		cache, err = loadInboxCache(cachePath)
		if err != nil {
			fmt.Fprintf(ios.ErrOut, "Warning: failed to load inbox cache: %v\n", err)
			cache = &inboxCache{Tasks: map[string]inboxCacheEntry{}}
		}
		if !cache.IsFresh(time.Now()) {
			cache = &inboxCache{Tasks: map[string]inboxCacheEntry{}}
		}
	}

	tasksToScan := cacheDiff(cache, workspaceTasks)
	if savings := len(workspaceTasks) - len(tasksToScan); savings > 0 {
		fmt.Fprintf(ios.ErrOut, "Cache hit: skipping %s (%d total, %d to fetch)\n",
			text.Pluralize(savings, "task"), len(workspaceTasks), len(tasksToScan))
	}

	// Phase 4: fetch comments concurrently for the diff only.
	if len(tasksToScan) > 0 {
		fmt.Fprintf(ios.ErrOut, "Checking comments on %s...\n", text.Pluralize(len(tasksToScan), "task"))
	}

	type taskComments struct {
		task     clickup.Task
		comments []commentData
		err      error
	}

	results := make([]taskComments, len(tasksToScan))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 15)

	for i, t := range tasksToScan {
		wg.Add(1)
		go func(idx int, task clickup.Task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			comments, err := fetchTaskComments(client, task.ID)
			results[idx] = taskComments{task: task, comments: comments, err: err}
		}(i, t)
	}
	wg.Wait()

	username := strings.ToLower(user.User.Username)

	// Phase 5a: build comment + description mention events from the comment scan.
	// Track mentions per task so we can persist them in the cache for warm runs.
	var mentionEvents []mention
	mentionsByTask := map[string][]mention{}
	addMention := func(m mention) {
		mentionEvents = append(mentionEvents, m)
		mentionsByTask[m.TaskID] = append(mentionsByTask[m.TaskID], m)
	}
	for _, r := range results {
		if r.err != nil {
			continue
		}

		desc := r.task.TextContent
		if desc == "" {
			desc = r.task.Description
		}
		if desc != "" && containsMention(desc, username) {
			ms, _ := strconv.ParseInt(r.task.DateUpdated, 10, 64)
			addMention(mention{
				TaskID:        r.task.ID,
				TaskName:      r.task.Name,
				Type:          eventDescriptionMention,
				CommentText:   desc,
				Author:        r.task.Creator.Username,
				Date:          r.task.DateUpdated,
				DateMs:        ms,
				IsDescription: true,
			})
		}

		for _, c := range r.comments {
			if containsMention(c.CommentText, username) && strings.ToLower(c.User.Username) != username {
				ms, _ := strconv.ParseInt(c.Date, 10, 64)
				attachments := extractAttachmentURLs(c.Comment)
				addMention(mention{
					TaskID:      r.task.ID,
					TaskName:    r.task.Name,
					Type:        eventCommentMention,
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

	// Replay mentions stored in the cache for tasks we skipped this run.
	replayed := cachedMentionsFor(cache, skippedTasks(cache, workspaceTasks))

	// Phase 5b: build assignment events from the assigned task list.
	var assignmentEvents []mention
	for _, t := range assignedTasks {
		if !isNewAssignment(t, user.User.ID, cutoffMs) {
			continue
		}
		ms, _ := strconv.ParseInt(t.DateCreated, 10, 64)
		assignmentEvents = append(assignmentEvents, mention{
			TaskID:   t.ID,
			TaskName: t.Name,
			Type:     eventAssignment,
			Author:   t.Creator.Username,
			Date:     t.DateCreated,
			DateMs:   ms,
		})
	}

	events := mergeEvents(assignmentEvents, mentionEvents, replayed)

	// Phase 6: persist updated cache. We merge replayed mentions back in so
	// that warm-run skipped tasks retain their mentions on the next save.
	// --no-cache skips the read but still writes, so the next run benefits.
	for _, m := range replayed {
		if _, refreshed := mentionsByTask[m.TaskID]; refreshed {
			// Task was rescanned this run; trust the fresh data.
			continue
		}
		mentionsByTask[m.TaskID] = append(mentionsByTask[m.TaskID], m)
	}
	if cache == nil {
		cache = &inboxCache{Tasks: map[string]inboxCacheEntry{}}
	}
	updateCacheFromTasks(cache, workspaceTasks, mentionsByTask, time.Now())
	if err := saveInboxCache(cachePath, cache); err != nil {
		fmt.Fprintf(ios.ErrOut, "Warning: failed to save inbox cache: %v\n", err)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, events)
	}

	if len(events) == 0 {
		fmt.Fprintf(ios.Out, "No new mentions or assignments in the last %s.\n", text.Pluralize(opts.days, "day"))
		return nil
	}

	fmt.Fprintf(ios.ErrOut, "\nShowing %s from the last %s\n\n",
		text.Pluralize(len(events), "event"),
		text.Pluralize(opts.days, "day"))

	for i, m := range events {
		action := "commented on"
		switch m.Type {
		case eventAssignment:
			action = "assigned this task to you"
		case eventDescriptionMention:
			action = "mentioned you in description of"
		}

		fmt.Fprintf(ios.Out, "%s %s %s  %s\n",
			cs.Bold(m.Author),
			action,
			cs.Cyan("#"+m.TaskID),
			cs.Gray(formatMentionDate(m.Date)),
		)
		fmt.Fprintf(ios.Out, "%s\n", m.TaskName)
		if m.CommentText != "" {
			fmt.Fprintf(ios.Out, "\n  %s\n", strings.TrimSpace(m.CommentText))
		}
		for _, url := range m.Attachments {
			fmt.Fprintf(ios.Out, "  %s %s\n", cs.Gray("attachment:"), url)
		}
		if i < len(events)-1 {
			fmt.Fprintln(ios.Out)
		}
	}

	fmt.Fprintln(ios.Out)
	fmt.Fprintln(ios.Out, cs.Gray("---"))
	fmt.Fprintln(ios.Out, cs.Gray("Quick actions:"))
	fmt.Fprintf(ios.Out, "  %s  clickup comment add <task-id> \"@user text\" (supports @mentions)\n", cs.Gray("Reply:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task view <task-id>\n", cs.Gray("View:"))
	fmt.Fprintf(ios.Out, "  %s  clickup inbox --no-cache\n", cs.Gray("Refresh:"))
	fmt.Fprintf(ios.Out, "  %s  clickup inbox --json\n", cs.Gray("JSON:"))

	return nil
}

// fetchTeamTasksLocal paginates GetFilteredTeamTasksLocal until it has at least `limit`
// tasks or the API runs out of pages. Returns at most `limit` tasks.
func fetchTeamTasksLocal(ctx context.Context, client *api.Client, teamID string, params apiv2.FilteredTeamTasksParams, limit int) ([]clickup.Task, error) {
	var all []clickup.Task
	for page := 0; len(all) < limit; page++ {
		params.Page = page
		batch, err := apiv2.GetFilteredTeamTasksLocal(ctx, client, teamID, params)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func getCurrentUser(client *api.Client) (*userResponse, error) {
	req, err := http.NewRequest("GET", client.URL("user"), nil)
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
	commentURL := client.URL("task/%s/comment", taskID)
	req, err := http.NewRequestWithContext(context.Background(), "GET", commentURL, nil)
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
