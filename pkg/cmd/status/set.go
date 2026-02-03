package status

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/git"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type setOptions struct {
	factory      *cmdutil.Factory
	targetStatus string
	taskID       string
}

// NewCmdSet returns the "status set" command.
func NewCmdSet(f *cmdutil.Factory) *cobra.Command {
	opts := &setOptions{
		factory: f,
	}

	cmd := &cobra.Command{
		Use:   "set <status> [task]",
		Short: "Set the status of a task",
		Long: `Change a task's status using fuzzy matching.

The STATUS argument is matched against available statuses for the task's space.
Matching priority: exact match, then case-insensitive contains, then fuzzy match.

If TASK is not provided, the task ID is auto-detected from the current git branch.`,
		Example: `  # Set status using auto-detected task from branch
  clickup status set "in progress"

  # Set status for a specific task
  clickup status set "done" CU-abc123

  # Fuzzy matching works too
  clickup status set "prog" CU-abc123`,
		Args:               cobra.RangeArgs(1, 2),
		PersistentPreRunE:  cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.targetStatus = args[0]
			if len(args) > 1 {
				opts.taskID = args[1]
			}
			return setRun(opts)
		},
	}

	return cmd
}

// spaceStatusResponse represents the response from GET /space/{id} containing statuses.
type spaceStatusResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Statuses []struct {
		ID         string `json:"id"`
		Status     string `json:"status"`
		Color      string `json:"color"`
		Type       string `json:"type"`
		Orderindex int    `json:"orderindex"`
	} `json:"statuses"`
}

func setRun(opts *setOptions) error {
	ios := opts.factory.IOStreams
	cs := ios.ColorScheme()

	// Resolve task ID.
	taskID := opts.taskID
	if taskID == "" {
		gitCtx, err := opts.factory.GitContext()
		if err != nil {
			return fmt.Errorf("failed to detect git context: %w\n\n%s", err, git.BranchNamingSuggestion(""))
		}
		if gitCtx.TaskID == nil {
			return fmt.Errorf("no task ID provided and none detected from branch\n\n%s", git.BranchNamingSuggestion(gitCtx.Branch))
		}
		taskID = gitCtx.TaskID.ID
		fmt.Fprintf(ios.ErrOut, "Detected task %s from branch %s\n", cs.Cyan(taskID), cs.Cyan(gitCtx.Branch))
	}

	client, err := opts.factory.ApiClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Fetch the task to determine its space.
	task, _, err := client.Clickup.Tasks.GetTask(ctx, taskID, nil)
	if err != nil {
		return fmt.Errorf("failed to get task %s: %w", taskID, err)
	}

	currentStatus := task.Status.Status

	// Fetch statuses for the task's space.
	spaceID := task.Space.ID
	spaceURL := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s", spaceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spaceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch space statuses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch space statuses (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var spaceResp spaceStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&spaceResp); err != nil {
		return fmt.Errorf("failed to parse space response: %w", err)
	}

	if len(spaceResp.Statuses) == 0 {
		return fmt.Errorf("no statuses found for space %s", spaceID)
	}

	// Collect available status names.
	statusNames := make([]string, len(spaceResp.Statuses))
	for i, s := range spaceResp.Statuses {
		statusNames[i] = s.Status
	}

	// Find the best matching status.
	matched, err := matchStatus(opts.targetStatus, statusNames)
	if err != nil {
		return err
	}

	// Update the task status.
	updateURL := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s", taskID)
	payload := map[string]string{"status": matched}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal update payload: %w", err)
	}

	updateReq, err := http.NewRequestWithContext(ctx, http.MethodPut, updateURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}
	updateReq.Header.Set("Content-Type", "application/json")

	updateResp, err := client.HTTPClient.Do(updateReq)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		return fmt.Errorf("failed to update task status (HTTP %d): %s", updateResp.StatusCode, string(body))
	}

	// Report success.
	fromColor := cs.StatusColor(strings.ToLower(currentStatus))
	toColor := cs.StatusColor(strings.ToLower(matched))
	fmt.Fprintf(ios.Out, "Status changed: %s %s %s\n",
		fromColor(fmt.Sprintf("'%s'", currentStatus)),
		cs.Gray("\u2192"),
		toColor(fmt.Sprintf("'%s'", matched)),
	)

	return nil
}

// matchStatus finds the best matching status from available statuses using a tiered strategy:
// 1. Exact match (case-insensitive)
// 2. Contains match (case-insensitive)
// 3. Fuzzy match using normalized fold ranking
func matchStatus(target string, available []string) (string, error) {
	targetLower := strings.ToLower(target)

	// Tier 1: Exact match (case-insensitive).
	for _, s := range available {
		if strings.ToLower(s) == targetLower {
			return s, nil
		}
	}

	// Tier 2: Contains match (case-insensitive).
	var containsMatches []string
	for _, s := range available {
		if strings.Contains(strings.ToLower(s), targetLower) {
			containsMatches = append(containsMatches, s)
		}
	}
	if len(containsMatches) == 1 {
		return containsMatches[0], nil
	}
	if len(containsMatches) > 1 {
		// If multiple contains matches, pick the shortest (most specific).
		best := containsMatches[0]
		for _, m := range containsMatches[1:] {
			if len(m) < len(best) {
				best = m
			}
		}
		return best, nil
	}

	// Tier 3: Fuzzy match using RankMatchNormalizedFold.
	type ranked struct {
		name string
		rank int
	}
	var fuzzyMatches []ranked
	for _, s := range available {
		rank := fuzzy.RankMatchNormalizedFold(target, s)
		if rank >= 0 {
			fuzzyMatches = append(fuzzyMatches, ranked{name: s, rank: rank})
		}
	}

	if len(fuzzyMatches) > 0 {
		// Pick the match with the best (lowest) rank.
		best := fuzzyMatches[0]
		for _, m := range fuzzyMatches[1:] {
			if m.rank < best.rank {
				best = m
			}
		}
		return best.name, nil
	}

	// No match found.
	return "", fmt.Errorf("no matching status found for %q\n\nAvailable statuses: %s",
		target, strings.Join(available, ", "))
}
