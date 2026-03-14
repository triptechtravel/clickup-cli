package cmdutil

import (
	"fmt"
	"strconv"

	"github.com/raksul/go-clickup/clickup"
	"github.com/triptechtravel/clickup-cli/internal/config"
)

// CustomIDTaskOptions returns GetTaskOptions configured for custom task ID
// resolution. When isCustomID is true, the options include both the
// CustomTaskIDs flag and the TeamID from the workspace config, which the
// ClickUp API requires together.
//
// Returns nil when isCustomID is false (standard task IDs need no options).
func CustomIDTaskOptions(cfg *config.Config, isCustomID bool) *clickup.GetTaskOptions {
	if !isCustomID {
		return nil
	}
	opts := &clickup.GetTaskOptions{
		CustomTaskIDs: true,
	}
	if cfg != nil && cfg.Workspace != "" {
		if tid, err := strconv.Atoi(cfg.Workspace); err == nil {
			opts.TeamID = tid
		}
	}
	return opts
}

// CustomIDTaskOptionsWithSubtasks returns GetTaskOptions for custom task ID
// resolution with subtask inclusion enabled.
func CustomIDTaskOptionsWithSubtasks(cfg *config.Config, isCustomID bool) *clickup.GetTaskOptions {
	opts := CustomIDTaskOptions(cfg, isCustomID)
	if opts == nil {
		opts = &clickup.GetTaskOptions{}
	}
	opts.IncludeSubTasks = true
	return opts
}

// CustomIDQueryParam returns the URL query string fragment for custom task
// ID resolution. Used by commands that build raw HTTP requests (e.g. comments).
//
// Returns "" when isCustomID is false, or "?custom_task_ids=true&team_id=..."
// when true.
func CustomIDQueryParam(cfg *config.Config, isCustomID bool) string {
	if !isCustomID {
		return ""
	}
	q := "?custom_task_ids=true"
	if cfg != nil && cfg.Workspace != "" {
		q += fmt.Sprintf("&team_id=%s", cfg.Workspace)
	}
	return q
}
