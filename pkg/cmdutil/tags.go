package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/triptechtravel/clickup-cli/internal/api"
)

// spaceTagsResponse represents the response from GET /space/{id}/tag.
type spaceTagsResponse struct {
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

// FetchSpaceTags fetches the available tag names for a ClickUp space.
func FetchSpaceTags(client *api.Client, spaceID string) ([]string, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/tag", spaceID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch space tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch space tags (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tagsResp spaceTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("failed to parse tags response: %w", err)
	}

	names := make([]string, len(tagsResp.Tags))
	for i, t := range tagsResp.Tags {
		names[i] = t.Name
	}
	return names, nil
}

// ValidateTags validates tag names against the available tags for a space.
// Unknown tags are warned about and filtered out. Only valid tags are returned.
// If validation cannot be performed (e.g. network error), all tags are returned unchanged.
func ValidateTags(client *api.Client, spaceID string, tags []string, w io.Writer) []string {
	available, err := FetchSpaceTags(client, spaceID)
	if err != nil || len(available) == 0 {
		return tags // graceful fallback
	}

	availableSet := make(map[string]bool, len(available))
	for _, t := range available {
		availableSet[strings.ToLower(t)] = true
	}

	var valid []string
	var unknown []string
	for _, tag := range tags {
		if availableSet[strings.ToLower(tag)] {
			valid = append(valid, tag)
		} else {
			unknown = append(unknown, tag)
		}
	}

	if len(unknown) > 0 {
		fmt.Fprintf(w, "Warning: unknown tag(s) %s (available: %s)\n",
			strings.Join(unknown, ", "),
			strings.Join(available, ", "))
	}

	return valid
}
