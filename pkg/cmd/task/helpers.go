package task

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/internal/api"
)

// parseDate converts a YYYY-MM-DD string to a *clickup.Date.
func parseDate(s string) (*clickup.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q (use YYYY-MM-DD format): %w", s, err)
	}
	return clickup.NewDate(t), nil
}

// parseDuration converts a human-readable duration string (e.g. "2h", "30m", "1h30m")
// to milliseconds.
func parseDuration(s string) (int, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q (use format like 2h, 30m, 1h30m): %w", s, err)
	}
	return int(d.Milliseconds()), nil
}

// setTaskPoints sets sprint/story points on a task via the ClickUp API.
// The go-clickup library's TaskRequest does not support points, so this
// makes a raw HTTP PUT request.
func setTaskPoints(client *api.Client, taskID string, points float64) error {
	body := fmt.Sprintf(`{"points":%g}`, points)
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.clickup.com/api/v2/task/%s", taskID), strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to set points: status %d", resp.StatusCode)
	}
	return nil
}
