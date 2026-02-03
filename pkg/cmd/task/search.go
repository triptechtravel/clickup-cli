package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/iostreams"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type searchOptions struct {
	factory   *cmdutil.Factory
	query     string
	space     string
	folder    string
	pick      bool
	comments  bool
	jsonFlags cmdutil.JSONFlags
}

type searchTask struct {
	ID       string `json:"id"`
	CustomID string `json:"custom_id"`
	Name     string `json:"name"`
	Status   struct {
		Status string `json:"status"`
	} `json:"status"`
	Priority struct {
		Priority string `json:"priority"`
	} `json:"priority"`
	Assignees []struct {
		Username string `json:"username"`
	} `json:"assignees"`
	URL string `json:"url"`
}

type searchResponse struct {
	Tasks []searchTask `json:"tasks"`
}

// matchKind describes how a task matched the query.
type matchKind int

const (
	matchSubstring matchKind = iota // exact substring match (highest priority)
	matchFuzzy                      // fuzzy match by name
	matchComment                    // matched via comment text
)

// scoredTask wraps a searchTask with match metadata for sorting.
type scoredTask struct {
	searchTask
	kind      matchKind
	fuzzyRank int // lower is better; only meaningful for matchFuzzy
}

// scoreTaskName checks whether a task name matches the query and returns the
// match kind and fuzzy rank. Returns ok=false if there is no match at all.
func scoreTaskName(query, name string) (kind matchKind, rank int, ok bool) {
	lowerName := strings.ToLower(name)
	lowerQuery := strings.ToLower(query)

	if strings.Contains(lowerName, lowerQuery) {
		return matchSubstring, 0, true
	}

	rank = fuzzy.RankMatchNormalizedFold(query, lowerName)
	if rank > -1 {
		return matchFuzzy, rank, true
	}

	return 0, 0, false
}

// sortScoredTasks sorts scored results by relevance: exact substring matches
// first, then fuzzy matches sorted by rank (ascending), then comment matches.
func sortScoredTasks(tasks []scoredTask) {
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].kind != tasks[j].kind {
			return tasks[i].kind < tasks[j].kind
		}
		// Within the same kind, sort fuzzy matches by rank.
		if tasks[i].kind == matchFuzzy {
			return tasks[i].fuzzyRank < tasks[j].fuzzyRank
		}
		return false
	})
}

// commentSearchResponse is the response from the ClickUp comments API.
type commentSearchResponse struct {
	Comments []struct {
		ID          string `json:"id"`
		CommentText string `json:"comment_text"`
		User        struct {
			Username string `json:"username"`
		} `json:"user"`
	} `json:"comments"`
}

// NewCmdSearch returns a command to search ClickUp tasks by name.
func NewCmdSearch(f *cmdutil.Factory) *cobra.Command {
	opts := &searchOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search tasks by name",
		Long: `Search ClickUp tasks across the workspace by name.

Returns tasks whose names match the search query using substring and fuzzy
matching. Exact substring matches are shown first, followed by fuzzy matches
sorted by relevance.

Use --space and --folder to narrow the search scope for faster results.
Use --comments to also search through task comments (slower).

In interactive mode (TTY), if many results are found you will be asked
whether to refine the search. Use --pick to interactively select a single
task and print only its ID.

When no exact match is found, the search automatically tries individual
words from the query and shows potentially related tasks.`,
		Example: `  # Search for tasks mentioning "payload"
  clickup task search payload

  # Search within a specific space
  clickup task search geozone --space Development

  # Search within a specific folder
  clickup task search nextjs --folder "Engineering sprint"

  # Also search through task comments
  clickup task search "migration issue" --comments

  # Interactively pick a task (prints selected task ID)
  clickup task search geozone --pick

  # JSON output
  clickup task search geozone --json`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.query = args[0]
			return runSearch(opts)
		},
	}

	cmd.Flags().StringVar(&opts.space, "space", "", "Limit search to a specific space (name or ID)")
	cmd.Flags().StringVar(&opts.folder, "folder", "", "Limit search to a specific folder (name, substring match)")
	cmd.Flags().BoolVar(&opts.pick, "pick", false, "Interactively select a task and print its ID")
	cmd.Flags().BoolVar(&opts.comments, "comments", false, "Also search through task comments (slower)")
	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runSearch(opts *searchOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()
	interactive := ios.IsTerminal() && !opts.jsonFlags.WantsJSON()

	// 90-second overall timeout to prevent hanging.
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	scored, err := doSearch(ctx, opts)
	if err != nil {
		return err
	}

	// Deduplicate by task ID (keep best match kind per task).
	scored = dedupScored(scored)

	// Sort by relevance.
	sortScoredTasks(scored)

	// Interactive: if too many results, offer to narrow down.
	if interactive && len(scored) > 15 {
		p := prompter.New(ios)
		fmt.Fprintf(ios.ErrOut, "Found %d results.\n", len(scored))
		refine, err := p.Confirm(fmt.Sprintf("Narrow down? (%d results)", len(scored)), true)
		if err != nil {
			return err
		}
		if refine {
			extra, err := p.Input("Add keywords to filter (applied to current results):", "")
			if err != nil {
				return err
			}
			if extra != "" {
				extra = strings.ToLower(extra)
				var filtered []scoredTask
				for _, t := range scored {
					if strings.Contains(strings.ToLower(t.Name), extra) {
						filtered = append(filtered, t)
					}
				}
				scored = filtered
				fmt.Fprintf(ios.ErrOut, "Narrowed to %d results.\n", len(scored))
			}
		}
	}

	// If no results, try splitting the query into individual words.
	if len(scored) == 0 {
		words := strings.Fields(opts.query)
		if len(words) > 1 {
			fmt.Fprintf(ios.ErrOut, "No exact match for %q, trying individual words...\n", opts.query)
			for _, word := range words {
				if len(word) < 3 {
					continue
				}
				wordOpts := *opts
				wordOpts.query = word
				wordTasks, err := doSearch(ctx, &wordOpts)
				if err != nil {
					continue
				}
				scored = append(scored, wordTasks...)
			}
			scored = dedupScored(scored)
			sortScoredTasks(scored)
			if len(scored) > 0 {
				fmt.Fprintf(ios.ErrOut, "Found %d potentially related tasks.\n", len(scored))
			}
		}
	}

	// Convert scored tasks back to plain tasks for output.
	allTasks := make([]searchTask, len(scored))
	matchKinds := make([]matchKind, len(scored))
	for i, s := range scored {
		allTasks[i] = s.searchTask
		matchKinds[i] = s.kind
	}

	if len(allTasks) == 0 {
		fmt.Fprintf(ios.ErrOut, "No tasks found matching %q\n", opts.query)
		if interactive {
			return noResultsPrompt(ios, opts)
		}
		return nil
	}

	// --pick mode: interactive selection.
	if opts.pick && interactive {
		return pickTask(ios, allTasks)
	}

	if opts.jsonFlags.WantsJSON() {
		return opts.jsonFlags.OutputJSON(ios.Out, allTasks)
	}

	tp := tableprinter.New(ios)
	tp.AddField(cs.Bold("ID"))
	tp.AddField(cs.Bold("NAME"))
	tp.AddField(cs.Bold("STATUS"))
	tp.AddField(cs.Bold("ASSIGNEE"))
	tp.AddField(cs.Bold("MATCH"))
	tp.EndRow()
	tp.SetTruncateColumn(1)

	for i, t := range allTasks {
		id := t.ID
		if t.CustomID != "" {
			id = t.CustomID
		}
		tp.AddField(id)
		tp.AddField(t.Name)

		statusFn := cs.StatusColor(strings.ToLower(t.Status.Status))
		tp.AddField(statusFn(t.Status.Status))

		var names []string
		for _, a := range t.Assignees {
			names = append(names, a.Username)
		}
		tp.AddField(strings.Join(names, ", "))

		// Show match type indicator.
		switch matchKinds[i] {
		case matchSubstring:
			tp.AddField(cs.Green("name"))
		case matchFuzzy:
			tp.AddField(cs.Yellow("fuzzy"))
		case matchComment:
			tp.AddField(cs.Cyan("comment"))
		}
		tp.EndRow()
	}

	return tp.Render()
}

// doSearch performs the actual search using either the paginated team endpoint
// or the space/folder hierarchy (when --space or --folder is specified).
func doSearch(ctx context.Context, opts *searchOptions) ([]scoredTask, error) {
	ios := opts.factory.IOStreams

	client, err := opts.factory.ApiClient()
	if err != nil {
		return nil, err
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return nil, err
	}

	teamID := cfg.Workspace
	if teamID == "" {
		return nil, fmt.Errorf("workspace ID required. Set with 'clickup auth login'")
	}

	// If --space or --folder is specified, go directly to targeted search.
	if opts.space != "" || opts.folder != "" {
		return searchViaSpaces(ctx, opts)
	}

	// Search across multiple pages of the team tasks endpoint.
	query := strings.ToLower(opts.query)
	var allScored []scoredTask
	for page := 0; page < 10; page++ {
		if ctx.Err() != nil {
			fmt.Fprintf(ios.ErrOut, "Search timed out\n")
			break
		}

		apiURL := fmt.Sprintf(
			"https://api.clickup.com/api/v2/team/%s/task?include_closed=true&page=%d&order_by=updated&reverse=true",
			teamID, page,
		)

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.DoRequest(req)
		if err != nil {
			return nil, fmt.Errorf("API request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			break
		}

		var result searchResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		if len(result.Tasks) == 0 {
			break
		}

		// Score tasks by name (substring + fuzzy).
		nameMatched, unmatched := filterTasksByName(query, result.Tasks)
		allScored = append(allScored, nameMatched...)

		// If --comments is enabled, check comments on unmatched tasks.
		if opts.comments && len(unmatched) > 0 {
			limit := len(unmatched)
			if limit > 100 {
				limit = 100
			}
			fmt.Fprintf(ios.ErrOut, "  checking comments on %d tasks...\n", limit)
			commentMatches := searchTaskComments(ctx, client, query, unmatched[:limit])
			allScored = append(allScored, commentMatches...)
		}

		fmt.Fprintf(ios.ErrOut, "  scanned page %d (%d tasks)...\n", page+1, len(result.Tasks))
	}

	// If nothing found via pagination, fall back to space traversal.
	if len(allScored) == 0 {
		fmt.Fprintf(ios.ErrOut, "Falling back to space/folder search...\n")
		spaceTasks, err := searchViaSpaces(ctx, opts)
		if err != nil {
			return nil, err
		}
		allScored = append(allScored, spaceTasks...)
	}

	return allScored, nil
}

// filterTasksByName scores tasks by name and separates matched from unmatched.
func filterTasksByName(query string, tasks []searchTask) (matched []scoredTask, unmatched []searchTask) {
	for _, t := range tasks {
		kind, rank, ok := scoreTaskName(query, t.Name)
		if ok {
			matched = append(matched, scoredTask{
				searchTask: t,
				kind:       kind,
				fuzzyRank:  rank,
			})
		} else {
			unmatched = append(unmatched, t)
		}
	}
	return
}

// searchTaskComments checks task comments for the query string using the
// ClickUp API. Returns scored tasks that matched via comments.
func searchTaskComments(ctx context.Context, client *api.Client, query string, tasks []searchTask) []scoredTask {
	var results []scoredTask
	for _, t := range tasks {
		if ctx.Err() != nil {
			break
		}
		if taskMatchesComment(ctx, client, query, t.ID) {
			results = append(results, scoredTask{
				searchTask: t,
				kind:       matchComment,
			})
		}
	}
	return results
}

// taskMatchesComment fetches comments for a single task and returns true if
// any comment text contains the query (case-insensitive substring match).
func taskMatchesComment(ctx context.Context, client *api.Client, query, taskID string) bool {
	commentURL := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/comment", url.PathEscape(taskID))
	req, err := http.NewRequestWithContext(ctx, "GET", commentURL, nil)
	if err != nil {
		return false
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	var result commentSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	lowerQuery := strings.ToLower(query)
	for _, c := range result.Comments {
		if strings.Contains(strings.ToLower(c.CommentText), lowerQuery) {
			return true
		}
	}
	return false
}

// dedupScored deduplicates scored tasks by ID, keeping the entry with the
// best (lowest) match kind for each task.
func dedupScored(tasks []scoredTask) []scoredTask {
	best := make(map[string]scoredTask)
	order := make(map[string]int) // preserve first-seen order for stability
	idx := 0
	for _, t := range tasks {
		existing, exists := best[t.ID]
		if !exists {
			best[t.ID] = t
			order[t.ID] = idx
			idx++
		} else if t.kind < existing.kind {
			best[t.ID] = t
		} else if t.kind == existing.kind && t.kind == matchFuzzy && t.fuzzyRank < existing.fuzzyRank {
			best[t.ID] = t
		}
	}
	result := make([]scoredTask, 0, len(best))
	for _, t := range best {
		result = append(result, t)
	}
	// Sort by first-seen order to keep stable ordering before relevance sort.
	sort.Slice(result, func(i, j int) bool {
		return order[result[i].ID] < order[result[j].ID]
	})
	return result
}

func dedup(tasks []searchTask) []searchTask {
	seen := make(map[string]bool)
	var unique []searchTask
	for _, t := range tasks {
		if !seen[t.ID] {
			seen[t.ID] = true
			unique = append(unique, t)
		}
	}
	return unique
}

func noResultsPrompt(ios *iostreams.IOStreams, opts *searchOptions) error {
	p := prompter.New(ios)
	idx, err := p.Select("What would you like to do?", []string{
		"Enter a task ID manually",
		"Try a different search",
		"Cancel",
	})
	if err != nil || idx == 2 {
		return nil
	}
	if idx == 0 {
		taskID, err := p.Input("Task ID:", "")
		if err != nil {
			return err
		}
		if taskID != "" {
			fmt.Fprintln(ios.Out, taskID)
		}
		return nil
	}
	if idx == 1 {
		newQuery, err := p.Input("Search query:", "")
		if err != nil {
			return err
		}
		if newQuery != "" {
			opts.query = newQuery
			return runSearch(opts)
		}
	}
	return nil
}

func pickTask(ios *iostreams.IOStreams, allTasks []searchTask) error {
	p := prompter.New(ios)

	// Build options list for selection.
	options := make([]string, len(allTasks))
	for i, t := range allTasks {
		id := t.ID
		if t.CustomID != "" {
			id = t.CustomID
		}
		status := t.Status.Status
		options[i] = fmt.Sprintf("[%s] %s (%s)", id, t.Name, status)
	}
	options = append(options, "Enter task ID manually", "Cancel")

	idx, err := p.Select("Select a task:", options)
	if err != nil {
		return err
	}

	if idx == len(options)-1 { // Cancel
		return nil
	}
	if idx == len(options)-2 { // Manual entry
		taskID, err := p.Input("Task ID:", "")
		if err != nil {
			return err
		}
		if taskID != "" {
			fmt.Fprintln(ios.Out, taskID)
		}
		return nil
	}

	// Print selected task ID.
	t := allTasks[idx]
	id := t.ID
	if t.CustomID != "" {
		id = t.CustomID
	}
	fmt.Fprintln(ios.Out, id)
	return nil
}

func searchViaSpaces(ctx context.Context, opts *searchOptions) ([]scoredTask, error) {
	ios := opts.factory.IOStreams
	client, err := opts.factory.ApiClient()
	if err != nil {
		return nil, err
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return nil, err
	}

	teamID := cfg.Workspace

	// Get spaces.
	spaces, _, err := client.Clickup.Spaces.GetSpaces(ctx, teamID, false)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(opts.query)
	var results []scoredTask

	for _, space := range spaces {
		if ctx.Err() != nil {
			fmt.Fprintf(ios.ErrOut, "Search timed out during space traversal\n")
			break
		}

		// Filter by --space if provided (match by name or ID).
		if opts.space != "" {
			if !strings.EqualFold(space.Name, opts.space) && space.ID != opts.space {
				continue
			}
		}

		fmt.Fprintf(ios.ErrOut, "  searching space %q...\n", space.Name)

		// Get folders in space.
		folders, _, err := client.Clickup.Folders.GetFolders(ctx, space.ID, false)
		if err != nil {
			continue
		}

		var listIDs []string

		for _, folder := range folders {
			if ctx.Err() != nil {
				break
			}

			// Filter by --folder if provided (substring match, case-insensitive).
			if opts.folder != "" {
				if !strings.Contains(strings.ToLower(folder.Name), strings.ToLower(opts.folder)) {
					continue
				}
			}

			fmt.Fprintf(ios.ErrOut, "    folder %q...\n", folder.Name)
			lists, _, err := client.Clickup.Lists.GetLists(ctx, folder.ID, false)
			if err != nil {
				continue
			}
			for _, l := range lists {
				listIDs = append(listIDs, l.ID)
			}
		}

		// Also get folderless lists (only if no --folder filter).
		if opts.folder == "" {
			folderlessURL := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/list", url.PathEscape(space.ID))
			req, err := http.NewRequestWithContext(ctx, "GET", folderlessURL, nil)
			if err == nil {
				resp, err := client.DoRequest(req)
				if err == nil {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					var listResp struct {
						Lists []struct {
							ID string `json:"id"`
						} `json:"lists"`
					}
					if json.Unmarshal(body, &listResp) == nil {
						for _, l := range listResp.Lists {
							listIDs = append(listIDs, l.ID)
						}
					}
				}
			}
		}

		fmt.Fprintf(ios.ErrOut, "    scanning %d lists...\n", len(listIDs))

		// Search tasks in each list.
		for _, listID := range listIDs {
			if ctx.Err() != nil {
				break
			}

			taskURL := fmt.Sprintf("https://api.clickup.com/api/v2/list/%s/task?include_closed=true&page=0", url.PathEscape(listID))
			req, err := http.NewRequestWithContext(ctx, "GET", taskURL, nil)
			if err != nil {
				continue
			}

			resp, err := client.DoRequest(req)
			if err != nil {
				continue
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var taskResp searchResponse
			if json.Unmarshal(body, &taskResp) == nil {
				nameMatched, unmatched := filterTasksByName(query, taskResp.Tasks)
				results = append(results, nameMatched...)

				// If --comments is enabled, check comments on unmatched tasks.
				if opts.comments && len(unmatched) > 0 {
					limit := len(unmatched)
					if limit > 100 {
						limit = 100
					}
					fmt.Fprintf(ios.ErrOut, "      checking comments on %d tasks...\n", limit)
					commentMatches := searchTaskComments(ctx, client, query, unmatched[:limit])
					results = append(results, commentMatches...)
				}
			}
		}
	}

	return results, nil
}
