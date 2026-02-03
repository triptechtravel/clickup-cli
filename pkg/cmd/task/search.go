package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
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

// NewCmdSearch returns a command to search ClickUp tasks by name.
func NewCmdSearch(f *cmdutil.Factory) *cobra.Command {
	opts := &searchOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search tasks by name",
		Long: `Search ClickUp tasks across the workspace by name.

Returns tasks whose names match the search query. Use --space and --folder
to narrow the search scope for faster results.

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

	allTasks, err := doSearch(ctx, opts)
	if err != nil {
		return err
	}

	// Deduplicate by task ID.
	allTasks = dedup(allTasks)

	// Interactive: if too many results, offer to narrow down.
	if interactive && len(allTasks) > 15 {
		p := prompter.New(ios)
		fmt.Fprintf(ios.ErrOut, "Found %d results.\n", len(allTasks))
		refine, err := p.Confirm(fmt.Sprintf("Narrow down? (%d results)", len(allTasks)), true)
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
				var filtered []searchTask
				for _, t := range allTasks {
					if strings.Contains(strings.ToLower(t.Name), extra) {
						filtered = append(filtered, t)
					}
				}
				allTasks = filtered
				fmt.Fprintf(ios.ErrOut, "Narrowed to %d results.\n", len(allTasks))
			}
		}
	}

	// If no exact results, try splitting the query into individual words.
	if len(allTasks) == 0 {
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
				allTasks = append(allTasks, wordTasks...)
			}
			allTasks = dedup(allTasks)
			if len(allTasks) > 0 {
				fmt.Fprintf(ios.ErrOut, "Found %d potentially related tasks.\n", len(allTasks))
			}
		}
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
	tp.EndRow()
	tp.SetTruncateColumn(1)

	for _, t := range allTasks {
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
		tp.EndRow()
	}

	return tp.Render()
}

// doSearch performs the actual search using either the paginated team endpoint
// or the space/folder hierarchy (when --space or --folder is specified).
func doSearch(ctx context.Context, opts *searchOptions) ([]searchTask, error) {
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
	var allTasks []searchTask
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

		query := strings.ToLower(opts.query)
		for _, t := range result.Tasks {
			if strings.Contains(strings.ToLower(t.Name), query) {
				allTasks = append(allTasks, t)
			}
		}

		fmt.Fprintf(ios.ErrOut, "  scanned page %d (%d tasks)...\n", page+1, len(result.Tasks))
	}

	// If nothing found via pagination, fall back to space traversal.
	if len(allTasks) == 0 {
		fmt.Fprintf(ios.ErrOut, "Falling back to space/folder search...\n")
		spaceTasks, err := searchViaSpaces(ctx, opts)
		if err != nil {
			return nil, err
		}
		allTasks = append(allTasks, spaceTasks...)
	}

	return allTasks, nil
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

func searchViaSpaces(ctx context.Context, opts *searchOptions) ([]searchTask, error) {
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
	var results []searchTask

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
				for _, t := range taskResp.Tasks {
					if strings.Contains(strings.ToLower(t.Name), query) {
						results = append(results, t)
					}
				}
			}
		}
	}

	return results, nil
}
