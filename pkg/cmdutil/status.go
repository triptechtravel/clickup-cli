package cmdutil

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/apiv2"
)

// MatchStatus finds the best matching status from available statuses using a tiered strategy:
// 1. Exact match (case-insensitive)
// 2. Contains match (case-insensitive)
// 3. Fuzzy match using normalized fold ranking
func MatchStatus(target string, available []string) (string, error) {
	targetLower := strings.ToLower(target)

	// Pre-compute lowercased statuses once for all tiers.
	lowerAvail := make([]string, len(available))
	for i, s := range available {
		lowerAvail[i] = strings.ToLower(s)
	}

	// Tier 1: Exact match (case-insensitive).
	for i, lower := range lowerAvail {
		if lower == targetLower {
			return available[i], nil
		}
	}

	// Tier 2: Contains match (case-insensitive).
	var containsMatches []string
	for i, lower := range lowerAvail {
		if strings.Contains(lower, targetLower) {
			containsMatches = append(containsMatches, available[i])
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

// ValidateStatus validates a status string against the available statuses for a task's list,
// falling back to space-level statuses if the list has no custom overrides.
// If the status fuzzy-matches, it returns the matched value and prints a warning to w.
// If no match, it returns an error with available statuses. If validation cannot be
// performed (e.g. network error), the original status is returned unchanged.
func ValidateStatus(client *api.Client, spaceID, status string, w io.Writer) (string, error) {
	return ValidateStatusWithList(client, spaceID, "", status, w)
}

// ValidateStatusWithList validates a status string against the available statuses for a list,
// falling back to space-level statuses if the list has no custom status overrides.
func ValidateStatusWithList(client *api.Client, spaceID, listID, status string, w io.Writer) (string, error) {
	// Try list-level statuses first (lists can override space statuses).
	if listID != "" {
		statusNames, err := FetchListStatuses(client, listID)
		if err == nil && len(statusNames) > 0 {
			return matchAndReport(status, statusNames, w)
		}
	}

	// Fall back to space-level statuses.
	statusNames, err := FetchSpaceStatuses(client, spaceID)
	if err != nil || len(statusNames) == 0 {
		return status, nil // graceful fallback
	}

	return matchAndReport(status, statusNames, w)
}

// ValidateStatusFromLists validates a status string against pre-fetched list statuses,
// falling back to space-level statuses if no list statuses are provided.
func ValidateStatusFromLists(client *api.Client, spaceID string, listStatuses []string, status string, w io.Writer) (string, error) {
	if len(listStatuses) > 0 {
		return matchAndReport(status, listStatuses, w)
	}

	// Fall back to space-level statuses.
	statusNames, err := FetchSpaceStatuses(client, spaceID)
	if err != nil || len(statusNames) == 0 {
		return status, nil // graceful fallback
	}

	return matchAndReport(status, statusNames, w)
}

func matchAndReport(status string, statusNames []string, w io.Writer) (string, error) {
	matched, err := MatchStatus(status, statusNames)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(matched, status) {
		fmt.Fprintf(w, "Status %q matched to %q\n", status, matched)
	}
	return matched, nil
}

// FetchSpaceStatuses fetches the available status names for a ClickUp space.
func FetchSpaceStatuses(client *api.Client, spaceID string) ([]string, error) {
	var spaceResp spaceStatusResponse
	if err := apiv2.Do(context.Background(), client, "GET", fmt.Sprintf("space/%s", spaceID), nil, &spaceResp); err != nil {
		return nil, fmt.Errorf("failed to fetch space statuses: %w", err)
	}

	statusNames := make([]string, len(spaceResp.Statuses))
	for i, s := range spaceResp.Statuses {
		statusNames[i] = s.Status
	}
	return statusNames, nil
}

// FetchListStatuses fetches the available status names for a ClickUp list.
// Returns nil if the list doesn't have custom status overrides.
func FetchListStatuses(client *api.Client, listID string) ([]string, error) {
	list, err := apiv2.GetListLocal(context.Background(), client, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch list statuses: %w", err)
	}

	if len(list.Statuses) == 0 {
		return nil, nil
	}

	statusNames := make([]string, len(list.Statuses))
	for i, s := range list.Statuses {
		statusNames[i] = s.Status
	}
	return statusNames, nil
}
