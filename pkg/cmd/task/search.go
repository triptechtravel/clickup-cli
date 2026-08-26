package task

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
	"github.com/triptechtravel/clickup-cli/internal/config"
	"github.com/triptechtravel/clickup-cli/internal/iostreams"
	"github.com/triptechtravel/clickup-cli/internal/prompter"
	"github.com/triptechtravel/clickup-cli/internal/tableprinter"
	"github.com/triptechtravel/clickup-cli/internal/taskindex"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type searchOptions struct {
	factory         *cmdutil.Factory
	query           string
	space           string
	folder          string
	assignee        string
	pick            bool
	comments        bool
	exact           bool
	includeSubtasks bool
	noCache         bool
	// skipSync suppresses the index refresh for the per-word retry, which
	// re-reads an index the first pass has already synced.
	skipSync  bool
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
	URL         string `json:"url"`
	Parent      string `json:"parent"`
	DateUpdated string `json:"date_updated"` // epoch millis, as a string
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
priority: name substring > name fuzzy > description substring.

ClickUp has no server-side text search, so matching happens locally. To keep
that from meaning "paginate the workspace on every search", the CLI keeps an
index of the workspace under ~/.cache/clickup (override with CLICKUP_CACHE_DIR)
and tops it up with only what changed since the last run. The first search
builds the index and is slow; later ones read every task in the workspace for
about one request. The index is rebuilt weekly so that deleted and archived
tasks fall out of it.

Use --no-cache to skip the index and query the API directly. That path reads
only the most recently updated tasks and says so when it runs out of budget.
--comments and --assignee also bypass the index.

Use --space and --folder to search a specific part of the tree instead; this
reaches tasks of any age, at the cost of walking every list.

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

  # Include subtasks in results
  clickup task search "Phase 1" --include-subtasks

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
	cmd.Flags().BoolVar(&opts.includeSubtasks, "include-subtasks", false, "Include subtasks in search results")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "Bypass the local task index and query the API directly")
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

	res, err := doSearch(ctx, opts)
	if err != nil {
		return err
	}
	scored := res.tasks
	disclosures := res

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
				// The index was synced by the first pass; re-syncing it once
				// per word buys nothing and costs a request each time.
				wordOpts.skipSync = true
				wordRes, err := doSearch(ctx, &wordOpts)
				if err != nil {
					continue
				}
				disclosures.mergeDisclosures(wordRes)
				scored = append(scored, wordRes.tasks...)
			}
			scored = dedupScored(scored)
			sortScoredTasks(scored)
			if len(scored) > 0 {
				fmt.Fprintf(ios.ErrOut, "Found %d potentially related tasks.\n", len(scored))
			}
		}
	}

	// Print every limit that bit, naming the mechanism rather than a generic
	// one: an absence of rows must never read as an absence of tasks, and a
	// disclosure that names the wrong cap sends the reader after the wrong fix.
	for _, n := range disclosures.notShown {
		fmt.Fprintf(ios.ErrOut, "Not shown: %s\n", n)
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
	// Newest first. ClickUp reads reverse=true on order_by=updated as *ascending*,
	// so asking for it pointed the sweep at the oldest tasks in the workspace —
	// and since the sweep only ever reads a prefix, the tasks people are working
	// on were the ones it could never reach.
	path := fmt.Sprintf("team/%s/task?include_closed=true&page=%d&order_by=updated",
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

// tasksPerPage is ClickUp's nominal page size for GET team/{id}/task. It is
// nominal only: a live page 0 comes back with 99 rows, so a short page proves
// nothing about the corpus and only an empty one ends the sweep. Used for
// sizing the comment probe and for reporting the reach of the page cap.
const tasksPerPage = 100

// sweepResult is what one paginated pass produced, together with whether the
// page cap cut it short.
type sweepResult struct {
	tasks []scoredTask
	// truncated means the read ran out of budget with more still behind it.
	truncated bool
	// cancelled means the deadline or an interrupt ended the read. Kept apart
	// from truncated so the user is not told the sweep "stopped at its page
	// cap" when it in fact stopped after one page.
	cancelled bool
	// notShown carries the disclosures to print. Held as text rather than
	// flags because several different limits can bite — the sweep's page cap,
	// the index sync budget, a list the space walk could not read — and naming
	// the wrong one sends the user after the wrong remedy.
	notShown []string
}

func (r *sweepResult) discloseF(format string, args ...any) {
	r.notShown = append(r.notShown, fmt.Sprintf(format, args...))
}

// mergeDisclosures carries another result's disclosures into this one, so a
// fallback path cannot quietly drop what the path before it had to admit.
func (r *sweepResult) mergeDisclosures(other sweepResult) {
	r.truncated = r.truncated || other.truncated
	r.cancelled = r.cancelled || other.cancelled
	for _, n := range other.notShown {
		if !slices.Contains(r.notShown, n) {
			r.notShown = append(r.notShown, n)
		}
	}
}

// sweepPages pulls up to maxPages of tasks and filters them here, returning
// every match rather than stopping at the first page that yields one.
func sweepPages(ctx context.Context, client *api.Client, teamID, query, extraParams string, maxPages int, comments bool) (sweepResult, error) {
	var res sweepResult
	for page := 0; page < maxPages; page++ {
		if ctx.Err() != nil {
			res.cancelled = true
			res.discloseF("the search ran out of time after %d page(s); results are partial.", page)
			return res, nil
		}

		tasks, err := fetchTeamTasks(ctx, client, teamID, page, extraParams)
		if err != nil {
			return res, err
		}
		if len(tasks) == 0 {
			return res, nil
		}

		matched, unmatched := filterTasks(query, tasks)
		res.tasks = append(res.tasks, matched...)

		if comments && len(unmatched) > 0 {
			limit := len(unmatched)
			if limit > tasksPerPage {
				limit = tasksPerPage
			}
			res.tasks = append(res.tasks, searchTaskComments(ctx, client, query, unmatched[:limit])...)
		}

	}

	// Every page came back full and the budget ran out: there is more behind it.
	res.truncated = true
	res.discloseF("the live sweep stopped at its %d-page cap (~%d most recently updated tasks); older matches may exist. Drop --no-cache to search the local index, or use --space/--folder.",
		maxPages, maxPages*tasksPerPage)
	return res, nil
}

// resolveAssignee resolves a user input (name, username, numeric ID, or "me")
// to a numeric user ID and display name. It uses the workspace members list.
func resolveAssignee(ctx context.Context, client *api.Client, input string) (int, string, error) {
	members, err := fetchWorkspaceMembers(ctx, client)
	if err != nil {
		return 0, "", err
	}

	var currentUserID int
	if strings.EqualFold(input, "me") {
		id, err := cmdutil.GetCurrentUserID(client)
		if err != nil {
			return 0, "", fmt.Errorf("failed to get current user: %w", err)
		}
		currentUserID = id
	}

	return resolveAssigneeFromMembers(members, input, currentUserID)
}

// fetchWorkspaceMembers returns the flattened list of members across all teams.
func fetchWorkspaceMembers(ctx context.Context, client *api.Client) ([]clickup.TeamUser, error) {
	teams, err := apiv2.GetTeamsLocal(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workspace members: %w", err)
	}
	var members []clickup.TeamUser
	for _, team := range teams {
		for _, m := range team.Members {
			members = append(members, m.User)
		}
	}
	return members, nil
}

// resolveAssigneeFromMembers matches input (name, username, numeric ID, or
// "me") against a pre-fetched member list. If input is "me", currentUserID
// must be pre-resolved by the caller.
func resolveAssigneeFromMembers(members []clickup.TeamUser, input string, currentUserID int) (int, string, error) {
	if strings.EqualFold(input, "me") {
		for _, m := range members {
			if m.ID == currentUserID {
				return currentUserID, m.Username, nil
			}
		}
		return currentUserID, "me", nil
	}

	if id, err := strconv.Atoi(input); err == nil {
		for _, m := range members {
			if m.ID == id {
				return id, m.Username, nil
			}
		}
		return 0, "", fmt.Errorf("no workspace member found with ID %d", id)
	}

	for _, m := range members {
		if strings.EqualFold(m.Username, input) {
			return m.ID, m.Username, nil
		}
	}

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

// maxSweepPages bounds the live workspace sweep.
//
// ClickUp exposes no server-side text search: the `search=` param on
// GET team/{id}/task is accepted and then ignored, returning the same
// date-ordered page whatever the query. Every match is therefore found by
// pulling full pages and filtering them here, so the only cost lever is how
// many pages we pull. The cap keeps a cold search from walking an unbounded
// workspace; when it bites, sweepResult.truncated says so rather than letting
// an absence of rows read as "there is nothing else".
const maxSweepPages = 10

// cacheTTL is how long the local index goes before a full rebuild. Incremental
// syncs only ever add: a task deleted or archived upstream stays in the mirror
// until something reconciles it, and this is that something.
const cacheTTL = 7 * 24 * time.Hour

// syncBudget caps how long a sync may take before the search gives up on it and
// works with what it has. The rate limiter sleeps uninterruptibly for up to a
// minute when the quota is exhausted, so a large rebuild can otherwise consume
// the whole command deadline and return nothing.
const syncBudget = 60 * time.Second

// maxListPages bounds how deep the space walk reads a single list. Generous,
// because this is the path advertised as reaching everything, but not
// unbounded — a runaway list should not hang the search.
const maxListPages = 20

// maxFullSyncPages bounds a cold rebuild. Far larger than maxSweepPages
// because it is paid once a week rather than once a search.
const maxFullSyncPages = 200

// toEntry projects a fetched task into what the index keeps.
func toEntry(t searchTask, at time.Time) taskindex.Entry {
	names := make([]string, 0, len(t.Assignees))
	for _, a := range t.Assignees {
		names = append(names, a.Username)
	}
	updated, _ := strconv.ParseInt(t.DateUpdated, 10, 64)
	return taskindex.Entry{
		ID:          t.ID,
		CustomID:    t.CustomID,
		Name:        t.Name,
		Description: t.Description,
		Status:      t.Status.Status,
		Priority:    t.Priority.Priority,
		Parent:      t.Parent,
		Assignees:   names,
		URL:         t.URL,
		DateUpdated: updated,
		IndexedAt:   at.UnixMilli(),
	}
}

// fromEntry rebuilds enough of a task for matching and display.
func fromEntry(e taskindex.Entry) searchTask {
	var t searchTask
	t.ID = e.ID
	t.CustomID = e.CustomID
	t.Name = e.Name
	t.Description = e.Description
	t.Status.Status = e.Status
	t.Priority.Priority = e.Priority
	t.Parent = e.Parent
	t.URL = e.URL
	t.DateUpdated = strconv.FormatInt(e.DateUpdated, 10)
	for _, n := range e.Assignees {
		t.Assignees = append(t.Assignees, struct {
			Username string `json:"username"`
		}{Username: n})
	}
	return t
}

// syncResult is what one sync pass managed to fetch, and whether it saw the
// whole of what it asked for. A short slice and a complete one are otherwise
// indistinguishable, which is how a capped fetch comes to be mistaken for the
// whole workspace.
type syncResult struct {
	entries []taskindex.Entry
	// truncated means the sync ran out of pages or time with more behind it.
	truncated bool
	// failed means the API refused partway through. Kept apart from truncated:
	// a transient 5xx must not stamp the reconcile clock and flag the index
	// incomplete, which would lock in a degraded index for a week over one bad
	// response.
	failed bool
	// cancelled means the sync budget or the command deadline expired.
	cancelled bool
}

// fetchEntries pulls tasks page by page until a page comes back empty, which is
// the only trustworthy end-of-corpus signal ClickUp gives.
//
// It reports rather than returns a truncation: running out of pages or time is
// a partial answer, not a failure, and the entries already fetched are worth
// keeping. A hard API error is still an error, but the pages read before it are
// handed back too.
func fetchEntries(ctx context.Context, client *api.Client, teamID, extraParams string, maxPages int, at time.Time) (syncResult, error) {
	var res syncResult
	for page := 0; page < maxPages; page++ {
		if ctx.Err() != nil {
			res.truncated = true
			res.cancelled = true
			return res, nil
		}
		tasks, err := fetchTeamTasks(ctx, client, teamID, page, extraParams)
		if err != nil {
			res.failed = true
			return res, err
		}
		if len(tasks) == 0 {
			return res, nil
		}
		for _, t := range tasks {
			res.entries = append(res.entries, toEntry(t, at))
		}
	}
	res.truncated = true
	return res, nil
}

// searchViaIndex answers from the local mirror, topping it up first.
//
// The sync always requests subtasks so one cache serves both modes; whether
// they are shown is decided here, not at fetch time.
func searchViaIndex(ctx context.Context, opts *searchOptions, client *api.Client, teamID, query string) (sweepResult, error) {
	dir := config.CacheDir()
	now := time.Now()

	idx, err := taskindex.Load(dir, teamID)
	if err != nil {
		return sweepResult{}, err
	}

	var out sweepResult

	// skipSync is set for the per-word retry, which re-reads the same index it
	// just synced. Without it a three-word query synced three more times.
	if !opts.skipSync {
		out, err = syncIndex(ctx, opts, client, teamID, dir, idx, now)
		if err != nil {
			return out, err
		}
	}
	if idx.Partial {
		out.truncated = true
		out.discloseF("the local index does not yet cover the whole workspace; the next search continues building it. Use --space/--folder to search older tasks now, or --no-cache to sweep the API directly.")
	}

	entries := idx.All()
	// Map iteration is unordered; without this, equally-scored matches would
	// come out in a different order on every run. Newest first mirrors the
	// live sweep.
	sort.SliceStable(entries, func(a, b int) bool {
		if entries[a].DateUpdated != entries[b].DateUpdated {
			return entries[a].DateUpdated > entries[b].DateUpdated
		}
		return entries[a].ID < entries[b].ID
	})

	tasks := make([]searchTask, 0, len(entries))
	for _, e := range entries {
		if e.Parent != "" && !opts.includeSubtasks {
			continue
		}
		tasks = append(tasks, fromEntry(e))
	}

	matched, _ := filterTasks(query, tasks)
	out.tasks = matched
	return out, nil
}

// syncIndex brings the index up to date, reporting what it could not cover.
//
// It never returns an error for a sync that merely fell short: pages already
// read are worth keeping, and an index that exists is what stops the next run
// from starting over. Only a failure with nothing at all to fall back on is
// fatal.
func syncIndex(ctx context.Context, opts *searchOptions, client *api.Client, teamID, dir string, idx *taskindex.Index, now time.Time) (sweepResult, error) {
	ios := opts.factory.IOStreams
	var out sweepResult

	// The sync gets its own budget so a slow rebuild cannot eat the whole
	// command's deadline. Note this bounds the sync only in wall-clock terms if
	// the transport honours cancellation — see the rate limiter, which is why
	// that had to be made context-aware.
	syncCtx, cancel := context.WithTimeout(ctx, syncBudget)
	defer cancel()

	// An index holding nothing has nothing to be incremental about, whatever
	// its bookkeeping says — this is the state a failed first run leaves.
	full := idx.NeedsFullSync(now, cacheTTL) || idx.SyncedAt == 0

	var (
		res     syncResult
		syncErr error
		apply   func(*taskindex.Index) bool
	)

	if full {
		params := "subtasks=true"
		resumeFloor := idx.RebuildFloor
		switch {
		case resumeFloor > 0:
			// Continue below where the last segment stopped. This is what lets
			// a workspace too large to rebuild inside one budget converge over
			// consecutive searches rather than failing identically every week.
			params += fmt.Sprintf("&date_updated_lt=%d", resumeFloor)
			fmt.Fprintf(ios.ErrOut, "  continuing local index rebuild...\n")
		case len(idx.Entries) == 0:
			fmt.Fprintf(ios.ErrOut, "  building local index (first run, this one is slow)...\n")
		default:
			fmt.Fprintf(ios.ErrOut, "  rebuilding local index...\n")
		}

		res, syncErr = fetchEntries(syncCtx, client, teamID, params, maxFullSyncPages, now)
		floor := oldestUpdated(res.entries)
		incomplete := res.truncated || res.failed

		apply = func(ix *taskindex.Index) bool {
			if ix.RebuildStartedAt == 0 {
				ix.BeginRebuild(now)
			}
			if incomplete {
				return ix.AdvanceRebuild(res.entries, floor)
			}
			if !ix.CompleteRebuild(res.entries, now) {
				out.truncated = true
				out.discloseF("the index rebuild came back empty against a populated cache and was rejected rather than trusted; the cache was left as it was.")
			}
			return true
		}
		if incomplete {
			out.truncated = true
			out.discloseF("the workspace is larger than one rebuild pass; the index covers the most recent tasks and the next search continues it. Use --space/--folder for older tasks meanwhile.")
		}
	} else {
		fmt.Fprintf(ios.ErrOut, "  searching local index...\n")
		params := fmt.Sprintf("subtasks=true&date_updated_gt=%d", idx.SinceAt(now))
		res, syncErr = fetchEntries(syncCtx, client, teamID, params, maxFullSyncPages, now)

		apply = func(ix *taskindex.Index) bool {
			switch {
			case res.failed:
				return ix.MergeKeepingWatermark(res.entries)
			case res.truncated:
				// Holding the watermark protects the unread gap but cannot
				// catch up on its own, so ask for a rebuild.
				ix.MergeKeepingWatermark(res.entries)
				ix.RequestFullSync()
				return true
			default:
				return ix.Merge(res.entries)
			}
		}
		if res.truncated {
			out.truncated = true
			out.discloseF("more has changed since the last search than one sync could read; the index will be rebuilt on the next search.")
		}
	}

	// Apply under a lock against the current on-disk state. Two searches at
	// once would otherwise each write their own copy, and the loser's work —
	// possibly a complete rebuild — would vanish.
	if err := taskindex.Update(dir, teamID, func(ix *taskindex.Index) bool {
		changed := apply(ix)
		*idx = *ix
		return changed
	}); err != nil {
		fmt.Fprintf(ios.ErrOut, "warning: could not write task index: %v\n", err)
		// Still search what this process fetched.
		apply(idx)
	}

	if syncErr != nil {
		if len(idx.Entries) == 0 {
			return out, syncErr
		}
		fmt.Fprintf(ios.ErrOut, "warning: index sync incomplete (%v); searching what is cached\n", syncErr)
		out.truncated = true
		if full {
			out.discloseF("the index build stopped early (%v); tasks older than the part that was read are missing.", syncErr)
		} else {
			out.discloseF("the index could not be brought fully up to date (%v); recent changes may be missing.", syncErr)
		}
	}
	if res.cancelled {
		out.cancelled = true
		out.discloseF("the index sync hit its %s budget; it will continue on the next search.", syncBudget)
	}

	return out, nil
}

// oldestUpdated returns the smallest DateUpdated in a fetched segment, which is
// how far down the corpus that segment reached. Zero when nothing was fetched.
func oldestUpdated(entries []taskindex.Entry) int64 {
	var floor int64
	for _, e := range entries {
		if e.DateUpdated == 0 {
			continue
		}
		if floor == 0 || e.DateUpdated < floor {
			floor = e.DateUpdated
		}
	}
	return floor
}

// doSearch performs the actual search using progressive drill-down or
// the space/folder hierarchy (when --space or --folder is specified).
func doSearch(ctx context.Context, opts *searchOptions) (sweepResult, error) {
	ios := opts.factory.IOStreams

	client, err := opts.factory.ApiClient()
	if err != nil {
		return sweepResult{}, err
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		return sweepResult{}, err
	}

	teamID := cfg.Workspace
	if teamID == "" {
		return sweepResult{}, fmt.Errorf("workspace ID required. Set with 'clickup auth login'")
	}

	// Resolve --assignee to a numeric ID if provided.
	var assigneeParam string
	var assigneeName string
	if opts.assignee != "" {
		assigneeID, name, err := resolveAssignee(ctx, client, opts.assignee)
		if err != nil {
			return sweepResult{}, err
		}
		assigneeParam = fmt.Sprintf("assignees[]=%d", assigneeID)
		assigneeName = name
		fmt.Fprintf(ios.ErrOut, "  assignee: %s (ID: %d)\n", assigneeName, assigneeID)
	}

	// If --space or --folder is specified, go directly to targeted search.
	if opts.space != "" || opts.folder != "" {
		return searchViaSpaces(ctx, opts)
	}

	// Build extra params combining assignee filter and subtasks toggle if present.
	buildParams := func(base string) string {
		parts := make([]string, 0, 3)
		if base != "" {
			parts = append(parts, base)
		}
		if assigneeParam != "" {
			parts = append(parts, assigneeParam)
		}
		if opts.includeSubtasks {
			parts = append(parts, "subtasks=true")
		}
		return strings.Join(parts, "&")
	}

	// If --assignee with no query: fetch all tasks for that assignee.
	if opts.query == "" && assigneeParam != "" {
		fmt.Fprintf(ios.ErrOut, "  fetching tasks for %s...\n", assigneeName)
		tasks, err := fetchTeamTasks(ctx, client, teamID, 0, buildParams(""))
		if err != nil {
			return sweepResult{}, err
		}
		var scored []scoredTask
		for _, t := range tasks {
			scored = append(scored, scoredTask{searchTask: t, kind: matchSubstring})
		}
		return sweepResult{tasks: scored}, nil
	}

	query := strings.ToLower(opts.query)

	// The index reads the whole workspace for about one request; the live sweep
	// below reads a bounded prefix for ten. Only the cases the index cannot
	// serve fall through: --comments needs comment bodies it does not hold, and
	// --assignee is one of the few filters ClickUp genuinely applies server-side.
	if !opts.noCache && !opts.comments && opts.assignee == "" {
		res, err := searchViaIndex(ctx, opts, client, teamID, query)
		if err != nil {
			return res, err
		}
		// No automatic tree walk from here, whatever the index's coverage.
		//
		// A complete index covers the same tasks the walk would visit, so
		// walking to confirm an empty result is pure cost. And when the index
		// is incomplete the walk is worse than useless as a reflex: measured at
		// 234 requests and 112 seconds on this workspace, per zero-result
		// search, and the per-word retry multiplied it by the word count. The
		// honest move is to say the index is incomplete — which the disclosure
		// above does — and let the user spend that cost deliberately with
		// --space/--folder or --no-cache.
		return res, nil
	}

	// One pass over the workspace, filtered here. There is no cheaper place to
	// do it (see maxSweepPages) and no tier worth stopping at: the old
	// drill-down returned the moment any narrower slice produced a match, which
	// is how an exactly-matching card two pages in went missing while a weaker
	// match from the current sprint was reported as the whole answer.
	fmt.Fprintf(ios.ErrOut, "  searching workspace...\n")
	res, err := sweepPages(ctx, client, teamID, query, buildParams(""), maxSweepPages, opts.comments)
	if err != nil {
		return sweepResult{}, err
	}
	if len(res.tasks) > 0 {
		return res, nil
	}

	// Nothing in the recent window. Walk the space/folder tree, which reaches
	// tasks too old to surface in the sweep at all.
	fmt.Fprintf(ios.ErrOut, "Falling back to space/folder search...\n")
	walk, err := searchViaSpaces(ctx, opts)
	// The sweep's own limits still applied; dropping them here turned a
	// partial search into a confident "no such task".
	walk.mergeDisclosures(res)
	return walk, err
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

func searchViaSpaces(ctx context.Context, opts *searchOptions) (sweepResult, error) {
	ios := opts.factory.IOStreams
	client, err := opts.factory.ApiClient()
	if err != nil {
		if ctx.Err() != nil {
			// The clock ran out during discovery. That is a partial answer to
			// be disclosed, not an error to be raised: the caller has results
			// from earlier stages to show.
			out := sweepResult{cancelled: true}
			out.discloseF("the space walk ran out of time before it could list the spaces; results are partial.")
			return out, nil
		}
		return sweepResult{}, err
	}

	cfg, err := opts.factory.Config()
	if err != nil {
		if ctx.Err() != nil {
			// The clock ran out during discovery. That is a partial answer to
			// be disclosed, not an error to be raised: the caller has results
			// from earlier stages to show.
			out := sweepResult{cancelled: true}
			out.discloseF("the space walk ran out of time before it could list the spaces; results are partial.")
			return out, nil
		}
		return sweepResult{}, err
	}

	teamID := cfg.Workspace

	// Get spaces.
	spaces, err := apiv2.GetSpacesLocal(ctx, client, teamID, false)
	if err != nil {
		if ctx.Err() != nil {
			// The clock ran out during discovery. That is a partial answer to
			// be disclosed, not an error to be raised: the caller has results
			// from earlier stages to show.
			out := sweepResult{cancelled: true}
			out.discloseF("the space walk ran out of time before it could list the spaces; results are partial.")
			return out, nil
		}
		return sweepResult{}, err
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
	var unreadable atomic.Int64
	var capped atomic.Int64
	var fetchWg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	for i, listID := range allListIDs {
		fetchWg.Add(1)
		go func(idx int, lid string) {
			defer fetchWg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Every page of the list, not just the first. This is the path the
			// truncation notice and the help text point at as the one that
			// reaches everything; reading page 0 and stopping made that a lie
			// for any list holding more than a page of tasks.
			var tasks []searchTask
			listTruncated := true
			for page := 0; page < maxListPages; page++ {
				taskPath := fmt.Sprintf("list/%s/task?include_closed=true&page=%d", url.PathEscape(lid), page)
				if opts.includeSubtasks {
					taskPath += "&subtasks=true"
				}
				var taskResp searchResponse
				if err := apiv2.Do(ctx, client, "GET", taskPath, nil, &taskResp); err != nil {
					// A deadline is not a permission problem. Counting every
					// list left unvisited when the clock ran out as "could not
					// be read" reported broken lists by the hundred when the
					// truth was one expired budget.
					if ctx.Err() == nil {
						// Silence here is what made a partial walk look complete.
						unreadable.Add(1)
					}
					listTruncated = false
					break
				}
				if len(taskResp.Tasks) == 0 {
					listTruncated = false
					break
				}
				tasks = append(tasks, taskResp.Tasks...)
			}
			if listTruncated {
				capped.Add(1)
			}
			taskResp := searchResponse{Tasks: tasks}

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

	out := sweepResult{tasks: allScored}
	// A list that could not be read, or one deeper than the page budget, is a
	// hole in the result. Saying nothing about it is what let a partial walk
	// pass for a complete one.
	if n := unreadable.Load(); n > 0 {
		out.truncated = true
		out.discloseF("%d list(s) could not be read and were skipped; matches in them are missing.", n)
	}
	if n := capped.Load(); n > 0 {
		out.truncated = true
		out.discloseF("%d list(s) hold more than %d pages and were only read that far.", n, maxListPages)
	}
	if ctx.Err() != nil {
		out.cancelled = true
		out.discloseF("the space walk ran out of time before visiting every list; results are partial.")
	}
	return out, nil
}
