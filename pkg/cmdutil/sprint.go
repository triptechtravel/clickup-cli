package cmdutil

import (
	"context"
	"strconv"
	"time"

	"github.com/raksul/go-clickup/clickup"
)

// ResolveCurrentSprintListID finds the current sprint's list ID in the given folder.
// Returns ("", nil) if no sprint matches today's date.
func ResolveCurrentSprintListID(ctx context.Context, clickupClient *clickup.Client, folderID string) (string, error) {
	lists, _, err := clickupClient.Lists.GetLists(ctx, folderID, false)
	if err != nil {
		return "", err
	}

	now := time.Now()
	for _, l := range lists {
		start := ParseMSTimestamp(l.StartDate)
		due := ParseMSTimestamp(l.DueDate)
		if !start.IsZero() && !due.IsZero() && !now.Before(start) && !now.After(due) {
			return l.ID, nil
		}
	}

	return "", nil
}

// ParseMSTimestamp parses a millisecond Unix timestamp string into a time.Time.
func ParseMSTimestamp(ms string) time.Time {
	if ms == "" || ms == "0" {
		return time.Time{}
	}
	n, err := strconv.ParseInt(ms, 10, 64)
	if err != nil || n == 0 {
		return time.Time{}
	}
	return time.UnixMilli(n)
}
