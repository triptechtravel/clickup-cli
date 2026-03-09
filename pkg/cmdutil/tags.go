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

// CreateSpaceTag creates a new tag in a ClickUp space.
// POST /api/v2/space/{space_id}/tag with body {"tag":{"name":"tag-name"}}
func CreateSpaceTag(client *api.Client, spaceID, tagName string) error {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/tag", spaceID)

	payload := fmt.Sprintf(`{"tag":{"name":%q}}`, tagName)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create tag (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return nil
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

// EnsureTagsExist checks which tags already exist in the space and auto-creates
// any missing ones. Returns the full list of tag names (all guaranteed to exist).
// If the space tags cannot be fetched, tags are returned as-is (graceful fallback).
func EnsureTagsExist(client *api.Client, spaceID string, tags []string, w io.Writer) []string {
	available, err := FetchSpaceTags(client, spaceID)
	if err != nil {
		return tags // graceful fallback
	}

	availableSet := make(map[string]bool, len(available))
	for _, t := range available {
		availableSet[strings.ToLower(t)] = true
	}

	for _, tag := range tags {
		if !availableSet[strings.ToLower(tag)] {
			if err := CreateSpaceTag(client, spaceID, tag); err != nil {
				fmt.Fprintf(w, "Warning: failed to create tag %q: %v\n", tag, err)
			} else {
				fmt.Fprintf(w, "Created tag: %s\n", tag)
				availableSet[strings.ToLower(tag)] = true
			}
		}
	}

	return tags
}
