package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/triptechtravel/clickup-cli/internal/api"
)

// MatchStatus finds the best matching status from available statuses using a tiered strategy:
// 1. Exact match (case-insensitive)
// 2. Contains match (case-insensitive)
// 3. Fuzzy match using normalized fold ranking
func MatchStatus(target string, available []string) (string, error) {
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

// ValidateStatus validates a status string against the available statuses for a space.
// If the status fuzzy-matches, it returns the matched value and prints a warning to w.
// If no match, it returns an error with available statuses. If validation cannot be
// performed (e.g. network error), the original status is returned unchanged.
func ValidateStatus(client *api.Client, spaceID, status string, w io.Writer) (string, error) {
	statusNames, err := FetchSpaceStatuses(client, spaceID)
	if err != nil || len(statusNames) == 0 {
		return status, nil // graceful fallback
	}

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
	spaceURL := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s", spaceID)

	req, err := http.NewRequest(http.MethodGet, spaceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch space statuses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch space statuses (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var spaceResp spaceStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&spaceResp); err != nil {
		return nil, fmt.Errorf("failed to parse space response: %w", err)
	}

	statusNames := make([]string, len(spaceResp.Statuses))
	for i, s := range spaceResp.Statuses {
		statusNames[i] = s.Status
	}
	return statusNames, nil
}
