package task

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
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
	assignee  string
	pick      bool
	comments  bool
	exact     bool
	jsonFlags cmdutil.JSONFlags
}

type searchTask struct {
	ID          string `json:"id"`
	CustomID    string `json:"custom_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      struct {
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
	matchSubstring   matchKind = iota // exact substring match (highest priority)
	matchFuzzy                        // fuzzy match by name
	matchDescription                  // matched via description substring
	matchComment                      // matched via comment text
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
		Use:   "search [query]",
		Short: "Search tasks by name and description",
		Long: `Search ClickUp tasks across the workspace by name and description.

Returns tasks whose names or descriptions match the search query. Matching
priority: name substring > name fuzzy > description substring. When no
--space or --folder is specified, search uses progressive drill-down:
server-side search first, then sprint tasks, then your assigned tasks,
then configured space, then full workspace.

Use --space and --folder to narrow the search scope for faster results.
Use --comments to also search through task comments (slower).
Use --assignee to filter by team member (name, username, ID, or "me").

In interactive mode (TTY), if many results are found you will be asked
whether to refine the search. Use --pick to interactively select a single
task and print only its ID.

When no exact match is found, the search automatically tries individual
words from the query and shows potentially related tasks.

If search returns no results, use 'clickup task recent' to see your
recently updated tasks and discover which folders/lists to search in.`,
		Example: `  # Search for tasks mentioning "payload"
  clickup task search payload

  # Search within a specific space
  clickup task search geozone --space Development

  # Search within a specific folder
  clickup task search nextjs --folder "Engineering sprint"

  # Also search through task comments
  clickup task search "migration issue" --comments

  # Filter by assignee
  clickup task search --assignee me
  clickup task search "bug" --assignee "Isaac"
  clickup task search --assignee 54695018

  # Interactively pick a task (prints selected task ID)
  clickup task search geozone --pick

  # If search returns no results, find your active folders first
  clickup task recent
  clickup task search geozone --folder "Engineering Sprint"

  # JSON output
  clickup task search geozone --json`,
		Args:              cobra.RangeArgs(0, 1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.query = args[0]
			}
			if opts.query == "" && opts.assignee == "" {
				return fmt.Errorf("query or --assignee is required")
			}
			return runSearch(opts)
		},
	}

	cmd.Flags().StringVar(&opts.space, "space", "", "Limit search to a specific space (name or ID)")
	cmd.Flags().StringVar(&opts.folder, "folder", "", "Limit search to a specific folder (name, substring match)")
	cmd.Flags().StringVar(&opts.assignee, "assignee", "", "Filter by assignee (name, username, numeric ID, or \"me\")")
	cmd.Flags().BoolVar(&opts.pick, "pick", false, "Interactively select a task and print its ID")
	cmd.Flags().BoolVar(&opts.comments, "comments", false, "Also search through task comments (slower)")
	cmd.Flags().BoolVar(&opts.exact, "exact", false, "Only show exact substring matches (no fuzzy results)")
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

	// --exact: filter out fuzzy matches, keeping only substring and comment matches.
	if opts.exact {
		var exactOnly []scoredTask
		for _, s := range scored {
			if s.kind != matchFuzzy {
				exactOnly = append(exactOnly, s)
			}
		}
		scored = exactOnly
	}

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
		fmt.Fprintf(ios.ErrOut, "\nTip: run 'clickup task recent' to see your recently updated tasks and discover active lists/folders.\n")
		fmt.Fprintf(ios.ErrOut, "     run 'clickup sprint current' to see your current sprint.\n")
		return nil
	}

	// When not using --exact, show a separator if there are no substring matches but fuzzy results exist.
	hasFuzzyOnly := false
	if !opts.exact {
		hasExact := false
		for _, mk := range matchKinds {
			if mk == matchSubstring {
				hasExact = true
				break
			}
		}
		if !hasExact {
			for _, mk := range matchKinds {
				if mk == matchFuzzy {
					hasFuzzyOnly = true
					break
				}
			}
		}
		if hasFuzzyOnly {
			fmt.Fprintf(ios.ErrOut, "No exact matches. Showing fuzzy results (use --exact to suppress):\n")
		}
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
		case matchDescription:
			tp.AddField(cs.Blue("desc"))
		case matchComment:
			tp.AddField(cs.Cyan("comment"))
		}
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
	fmt.Fprintf(ios.Out, "  %s  clickup task edit <id> --status <status>\n", cs.Gray("Edit:"))
	fmt.Fprintf(ios.Out, "  %s  clickup sprint current\n", cs.Gray("Sprint:"))
	fmt.Fprintf(ios.Out, "  %s  clickup task search %q --json\n", cs.Gray("JSON:"), opts.query)

	return nil
}

// fetchTeamTasks fetches one page of tasks from the team endpoint with optional extra query params.
func fetchTeamTasks(ctx context.Context, client *api.Client, teamID string, page int, extraParams string) ([]searchTask, error) {
	path := fmt.Sprintf("team/%s/task?include_closed=true&page=%d&order_by=updated&reverse=true",
		teamID, page)
	if extraParams != "" {
		path += "&" + extraParams
	}

	var result searchResponse
	if err := apiv2.Do(ctx, client, "GET", path, nil, &result); err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	return result.Tasks, nil
}

// searchLevel searches tasks at a given drill-down level and returns scored matches.
func searchLevel(ctx context.Context, client *api.Client, teamID, query string, extraParams string, maxPages int, comments bool, ios *iostreams.IOStreams) ([]scoredTask, error) {
	var allScored []scoredTask
	for page := 0; page < maxPages; page++ {
		if ctx.Err() != nil {
			break
		}

		tasks, err := fetchTeamTasks(ctx, client, teamID, page, extraParams)
		if err != nil {
			return nil, err
		}
		if len(tasks) == 0 {
			break
		}

		matched, unmatched := filterTasks(query, tasks)
		allScored = append(allScored, matched...)

		if comments && len(unmatched) > 0 {
			limit := len(unmatched)
			if limit > 100 {
				limit = 100
			}
			commentMatches := searchTaskComments(ctx, client, query, unmatched[:limit])
			allScored = append(allScored, commentMatches...)
		}
	}
	return allScored, nil
}

// resolveAssignee resolves a user input (name, username, numeric ID, or "me")
// to a numeric user ID and display name. It uses the workspace members list.
func resolveAssignee(ctx context.Context, client *api.Client, input string) (int, string, error) {
	teams, err := apiv2.GetTeamsLocal(ctx, client)
	if err != nil {
		return 0, "", fmt.Errorf("failed to fetch workspace members: %w", err)
	}

	// Collect all members across teams.
	var members []clickup.TeamUser
	for _, team := range teams {
		for _, m := range team.Members {
			members = append(members, m.User)
		}
	}

	// "me" — resolve via current user ID, then look up display name.
	if strings.EqualFold(input, "me") {
		userID, err := cmdutil.GetCurrentUserID(client)
		if err != nil {
			return 0, "", fmt.Errorf("failed to get current user: %w", err)
		}
		for _, m := range members {
			if m.ID == userID {
				return userID, m.Username, nil
			}
		}
		return userID, "me", nil
	}

	// Numeric ID — parse and look up.
	if id, err := strconv.Atoi(input); err == nil {
		for _, m := range members {
			if m.ID == id {
				return id, m.Username, nil
			}
		}
		return 0, "", fmt.Errorf("no workspace member found with ID %d", id)
	}

	// Exact username match (case-insensitive).
	for _, m := range members {
		if strings.EqualFold(m.Username, input) {
			return m.ID, m.Username, nil
		}
	}

	// Substring match on username.
	lowerInput := strings.ToLower(input)
	var matches []clickup.TeamUser
	for _, m := range members {
		if strings.Contains(strings.ToLower(m.Username), lowerInput) {
			matches = append(matches, m)
		}
	}

	if len(matches) == 1 {
		return matches[0].ID, matches[0].Username, nil
	}
	if len(matches) > 1 {
		var names []string
		for _, m := range matches {
			names = append(names, fmt.Sprintf("%s (ID: %d)", m.Username, m.ID))
		}
		return 0, "", fmt.Errorf("ambiguous match, did you mean: %s", strings.Join(names, ", "))
	}

	return 0, "", fmt.Errorf("no workspace member found matching %q", input)
}

// doSearch performs the actual search using progressive drill-down or
// the space/folder hierarchy (when --space or --folder is specified).
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

	// Resolve --assignee to a numeric ID if provided.
	var assigneeParam string
	var assigneeName string
	if opts.assignee != "" {
		assigneeID, name, err := resolveAssignee(ctx, client, opts.assignee)
		if err != nil {
			return nil, err
		}
		assigneeParam = fmt.Sprintf("assignees[]=%d", assigneeID)
		assigneeName = name
		fmt.Fprintf(ios.ErrOut, "  assignee: %s (ID: %d)\n", assigneeName, assigneeID)
	}

	// If --space or --folder is specified, go directly to targeted search.
	if opts.space != "" || opts.folder != "" {
		return searchViaSpaces(ctx, opts)
	}

	// If --assignee with no query: fetch all tasks for that assignee.
	if opts.query == "" && assigneeParam != "" {
		fmt.Fprintf(ios.ErrOut, "  fetching tasks for %s...\n", assigneeName)
		tasks, err := fetchTeamTasks(ctx, client, teamID, 0, assigneeParam)
		if err != nil {
			return nil, err
		}
		var scored []scoredTask
		for _, t := range tasks {
			scored = append(scored, scoredTask{searchTask: t, kind: matchSubstring})
		}
		return scored, nil
	}

	query := strings.ToLower(opts.query)

	// Build extra params combining assignee filter if present.
	buildParams := func(base string) string {
		if assigneeParam == "" {
			return base
		}
		if base == "" {
			return assigneeParam
		}
		return base + "&" + assigneeParam
	}

	// Progressive drill-down: server-side → sprint → user → space → workspace.

	// Level 0: Server-side search (fastest — single API call).
	fmt.Fprintf(ios.ErrOut, "  searching (server-side)...\n")
	scored, err := searchLevel(ctx, client, teamID, query, buildParams("search="+url.QueryEscape(opts.query)), 1, opts.comments, ios)
	if err == nil && len(scored) > 0 {
		return scored, nil
	}

	// Level 1: Sprint list (if sprint_folder configured).
	if cfg.SprintFolder != "" {
		fmt.Fprintf(ios.ErrOut, "  searching sprint...\n")
		listID, err := cmdutil.ResolveCurrentSprintListID(ctx, client, cfg.SprintFolder)
		if err == nil && listID != "" {
			scored, err := searchLevel(ctx, client, teamID, query, buildParams("list_ids[]="+listID), 1, opts.comments, ios)
			if err != nil {
				return nil, err
			}
			if len(scored) > 0 {
				return scored, nil
			}
		}
	}

	// Level 2: User's assigned tasks.
	fmt.Fprintf(ios.ErrOut, "  searching your tasks...\n")
	userID, err := cmdutil.GetCurrentUserID(client)
	if err == nil {
		scored, err := searchLevel(ctx, client, teamID, query, buildParams(fmt.Sprintf("assignees[]=%d", userID)), 1, opts.comments, ios)
		if err != nil {
			return nil, err
		}
		if len(scored) > 0 {
			return scored, nil
		}
	}

	// Level 3: Configured space.
	if cfg.Space != "" {
		fmt.Fprintf(ios.ErrOut, "  searching space...\n")
		scored, err := searchLevel(ctx, client, teamID, query, buildParams("space_ids[]="+cfg.Space), 3, opts.comments, ios)
		if err != nil {
			return nil, err
		}
		if len(scored) > 0 {
			return scored, nil
		}
	}

	// Level 4: Full workspace (up to 10 pages).
	fmt.Fprintf(ios.ErrOut, "  searching workspace...\n")
	scored, err = searchLevel(ctx, client, teamID, query, buildParams(""), 10, opts.comments, ios)
	if err != nil {
		return nil, err
	}
	if len(scored) > 0 {
		return scored, nil
	}

	// Level 5: If nothing found via pagination, fall back to space traversal.
	fmt.Fprintf(ios.ErrOut, "Falling back to space/folder search...\n")
	return searchViaSpaces(ctx, opts)
}

// filterTasks scores tasks by name and description, separating matched from unmatched.
// Priority: name substring > name fuzzy > description substring.
func filterTasks(query string, tasks []searchTask) (matched []scoredTask, unmatched []searchTask) {
	lowerQuery := strings.ToLower(query)
	for _, t := range tasks {
		kind, rank, ok := scoreTaskName(query, t.Name)
		if ok {
			matched = append(matched, scoredTask{
				searchTask: t,
				kind:       kind,
				fuzzyRank:  rank,
			})
		} else if strings.Contains(strings.ToLower(t.Description), lowerQuery) {
			matched = append(matched, scoredTask{
				searchTask: t,
				kind:       matchDescription,
			})
		} else {
			unmatched = append(unmatched, t)
		}
	}
	return
}

// searchTaskComments checks task comments for the query string using the
// ClickUp API. Uses bounded concurrency (up to 5 parallel requests) to
// avoid serial N+1 API calls on large result sets.
func searchTaskComments(ctx context.Context, client *api.Client, query string, tasks []searchTask) []scoredTask {
	const maxWorkers = 5

	type result struct {
		task  searchTask
		match bool
	}

	// Fan out with bounded concurrency.
	sem := make(chan struct{}, maxWorkers)
	resultCh := make(chan result, len(tasks))
	var wg sync.WaitGroup

	for _, t := range tasks {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{} // acquire
		wg.Add(1)
		go func(task searchTask) {
			defer wg.Done()
			defer func() { <-sem }() // release
			match := taskMatchesComment(ctx, client, query, task.ID)
			resultCh <- result{task: task, match: match}
		}(t)
	}

	// Wait for all goroutines to finish.
	wg.Wait()
	close(resultCh)

	var scored []scoredTask
	for r := range resultCh {
		if r.match {
			scored = append(scored, scoredTask{
				searchTask: r.task,
				kind:       matchComment,
			})
		}
	}
	return scored
}

// taskMatchesComment fetches comments for a single task and returns true if
// any comment text contains the query (case-insensitive substring match).
func taskMatchesComment(ctx context.Context, client *api.Client, query, taskID string) bool {
	var result commentSearchResponse
	if err := apiv2.Do(ctx, client, "GET", fmt.Sprintf("task/%s/comment", url.PathEscape(taskID)), nil, &result); err != nil {
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
// best (lowest) match kind for each task. Preserves insertion order via
// index-based tracking (no extra sort pass needed since the caller runs
// sortScoredTasks immediately after).
func dedupScored(tasks []scoredTask) []scoredTask {
	bestIdx := make(map[string]int) // ID → index in result
	result := make([]scoredTask, 0, len(tasks)/2+1)
	for _, t := range tasks {
		if idx, exists := bestIdx[t.ID]; exists {
			existing := result[idx]
			if t.kind < existing.kind ||
				(t.kind == existing.kind && t.kind == matchFuzzy && t.fuzzyRank < existing.fuzzyRank) {
				result[idx] = t
			}
		} else {
			bestIdx[t.ID] = len(result)
			result = append(result, t)
		}
	}
	return result
}

func noResultsPrompt(ios *iostreams.IOStreams, opts *searchOptions) error {
	p := prompter.New(ios)
	idx, err := p.Select("What would you like to do?", []string{
		"Show my recent tasks",
		"Try a different search",
		"Enter a task ID manually",
		"Cancel",
	})
	if err != nil || idx == 3 {
		return nil
	}
	if idx == 0 {
		return showRecentTasksInteractive(ios, opts)
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
	if idx == 2 {
		taskID, err := p.Input("Task ID:", "")
		if err != nil {
			return err
		}
		if taskID != "" {
			fmt.Fprintln(ios.Out, taskID)
		}
		return nil
	}
	return nil
}

func showRecentTasksInteractive(ios *iostreams.IOStreams, opts *searchOptions) error {
	cs := ios.ColorScheme()

	fmt.Fprintln(ios.ErrOut, "Fetching your recent tasks...")
	tasks, err := cmdutil.FetchRecentTasks(opts.factory, 15)
	if err != nil {
		return fmt.Errorf("failed to fetch recent tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Fprintln(ios.ErrOut, "No recent tasks found.")
		return nil
	}

	// Show location context.
	locations := cmdutil.LocationSummary(tasks)
	if len(locations) > 0 {
		fmt.Fprintf(ios.ErrOut, "Your active locations: %s\n", strings.Join(locations, ", "))
		fmt.Fprintf(ios.ErrOut, "Tip: use %s to search within a specific folder.\n\n",
			cs.Bold("--folder \"name\""))
	}

	if opts.pick {
		// In pick mode, let user select a task.
		p := prompter.New(ios)
		options := make([]string, len(tasks))
		for i, t := range tasks {
			options[i] = cmdutil.FormatRecentTaskOption(t)
		}
		options = append(options, "Cancel")

		selected, err := p.Select("Select a task:", options)
		if err != nil || selected == len(options)-1 {
			return nil
		}
		fmt.Fprintln(ios.Out, tasks[selected].ID)
		return nil
	}

	// Display as table.
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

	return tp.Render()
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
	spaces, err := apiv2.GetSpacesLocal(ctx, client, teamID, false)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(opts.query)

	// Phase 1: Discover all list IDs across spaces (parallel per space).
	type spaceListIDs struct {
		listIDs []string
	}
	spaceResults := make([]spaceListIDs, len(spaces))
	var discoverWg sync.WaitGroup

	for i, space := range spaces {
		// Filter by --space if provided (match by name or ID).
		if opts.space != "" {
			if !strings.EqualFold(space.Name, opts.space) && space.ID != opts.space {
				continue
			}
		}

		discoverWg.Add(1)
		go func(idx int, sp clickup.Space) {
			defer discoverWg.Done()
			fmt.Fprintf(ios.ErrOut, "  searching space %q...\n", sp.Name)

			var listIDs []string

			// Folders and folderless lists concurrently within this space.
			var innerWg sync.WaitGroup
			var mu sync.Mutex

			// Folders.
			innerWg.Add(1)
			go func() {
				defer innerWg.Done()
				folders, err := apiv2.GetFoldersLocal(ctx, client, sp.ID, false)
				if err != nil {
					return
				}
				for _, folder := range folders {
					// Filter by --folder if provided.
					if opts.folder != "" {
						if !strings.Contains(strings.ToLower(folder.Name), strings.ToLower(opts.folder)) {
							continue
						}
					}
					lists, err := apiv2.GetListsLocal(ctx, client, folder.ID, false)
					if err != nil {
						continue
					}
					mu.Lock()
					for _, l := range lists {
						listIDs = append(listIDs, l.ID)
					}
					mu.Unlock()
				}
			}()

			// Folderless lists (only if no --folder filter).
			if opts.folder == "" {
				innerWg.Add(1)
				go func() {
					defer innerWg.Done()
					lists, err := apiv2.GetFolderlessListsLocal(ctx, client, sp.ID, false)
					if err != nil {
						return
					}
					mu.Lock()
					for _, l := range lists {
						listIDs = append(listIDs, l.ID)
					}
					mu.Unlock()
				}()
			}

			innerWg.Wait()
			spaceResults[idx] = spaceListIDs{listIDs: listIDs}
		}(i, space)
	}
	discoverWg.Wait()

	// Collect all list IDs.
	var allListIDs []string
	for _, sr := range spaceResults {
		allListIDs = append(allListIDs, sr.listIDs...)
	}

	fmt.Fprintf(ios.ErrOut, "    scanning %d lists...\n", len(allListIDs))

	// Phase 2: Fetch tasks from each list with bounded parallelism (5 workers).
	const maxWorkers = 5
	type listResult struct {
		scored []scoredTask
	}

	results := make([]listResult, len(allListIDs))
	var fetchWg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	for i, listID := range allListIDs {
		fetchWg.Add(1)
		go func(idx int, lid string) {
			defer fetchWg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			taskPath := fmt.Sprintf("list/%s/task?include_closed=true&page=0", url.PathEscape(lid))
			var taskResp searchResponse
			if err := apiv2.Do(ctx, client, "GET", taskPath, nil, &taskResp); err != nil {
				return
			}

			// If no query (assignee-only mode), return all tasks.
			if query == "" {
				var scored []scoredTask
				for _, t := range taskResp.Tasks {
					scored = append(scored, scoredTask{searchTask: t, kind: matchSubstring})
				}
				results[idx] = listResult{scored: scored}
				return
			}

			nameMatched, unmatched := filterTasks(query, taskResp.Tasks)
			var scored []scoredTask
			scored = append(scored, nameMatched...)

			// If --comments is enabled, check comments on unmatched tasks.
			if opts.comments && len(unmatched) > 0 {
				limit := len(unmatched)
				if limit > 100 {
					limit = 100
				}
				commentMatches := searchTaskComments(ctx, client, query, unmatched[:limit])
				scored = append(scored, commentMatches...)
			}
			results[idx] = listResult{scored: scored}
		}(i, listID)
	}
	fetchWg.Wait()

	// Collect all results.
	var allScored []scoredTask
	for _, r := range results {
		allScored = append(allScored, r.scored...)
	}

	return allScored, nil
}
