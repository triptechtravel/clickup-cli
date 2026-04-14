package cmdutil

import (
	"fmt"

	"github.com/triptechtravel/clickup-cli/internal/apiv2"
	"github.com/triptechtravel/clickup-cli/internal/config"
)

// CustomIDTaskQuery returns a query string for task endpoints supporting
// custom task IDs. Returns "" when isCustomID is false.
func CustomIDTaskQuery(cfg *config.Config, isCustomID bool) string {
	teamID := ""
	if cfg != nil {
		teamID = cfg.Workspace
	}
	return apiv2.TaskQuery(isCustomID, teamID, false)
}

// CustomIDTaskQueryWithSubtasks returns a query string for task endpoints
// supporting custom task IDs with subtask inclusion enabled.
func CustomIDTaskQueryWithSubtasks(cfg *config.Config, isCustomID bool) string {
	teamID := ""
	if cfg != nil {
		teamID = cfg.Workspace
	}
	return apiv2.TaskQuery(isCustomID, teamID, true)
}

// CustomIDTaskQueryMD returns a query string for task endpoints supporting
// custom task IDs with markdown description and subtask inclusion.
func CustomIDTaskQueryMD(cfg *config.Config, isCustomID bool) string {
	teamID := ""
	if cfg != nil {
		teamID = cfg.Workspace
	}
	return apiv2.TaskQueryMD(isCustomID, teamID)
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
