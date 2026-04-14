package apiv2

import (
	"context"
	"fmt"
	"net/url"

	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
)

// --- Query helpers ---

// TaskQuery builds a query string for task endpoints supporting custom task IDs
// and optional subtask inclusion.
func TaskQuery(customTaskIDs bool, teamID string, includeSubtasks bool) string {
	q := url.Values{}
	if customTaskIDs {
		q.Set("custom_task_ids", "true")
		if teamID != "" {
			q.Set("team_id", teamID)
		}
	}
	if includeSubtasks {
		q.Set("include_subtasks", "true")
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

// TaskQueryMD builds a query string like TaskQuery but also includes
// include_markdown_description=true.
func TaskQueryMD(customTaskIDs bool, teamID string) string {
	q := url.Values{}
	if customTaskIDs {
		q.Set("custom_task_ids", "true")
		if teamID != "" {
			q.Set("team_id", teamID)
		}
	}
	q.Set("include_markdown_description", "true")
	q.Set("include_subtasks", "true")
	return "?" + q.Encode()
}

// DependencyQuery builds a query string for dependency endpoints.
func DependencyQuery(dependsOn, dependencyOf string, customTaskIDs bool, teamID string) string {
	q := url.Values{}
	if dependsOn != "" {
		q.Set("depends_on", dependsOn)
	}
	if dependencyOf != "" {
		q.Set("dependency_of", dependencyOf)
	}
	if customTaskIDs {
		q.Set("custom_task_ids", "true")
	}
	if teamID != "" {
		q.Set("team_id", teamID)
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

// --- Tasks ---

// GetTaskLocal fetches a single task. qs is a query string (e.g. from TaskQuery).
func GetTaskLocal(ctx context.Context, client *api.Client, taskID, qs string) (*clickup.Task, error) {
	var task clickup.Task
	path := fmt.Sprintf("task/%s%s", taskID, qs)
	if err := do(ctx, client, "GET", path, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTasksLocal fetches tasks from a list. qs is a query string.
func GetTasksLocal(ctx context.Context, client *api.Client, listID, qs string) ([]clickup.Task, error) {
	var resp struct {
		Tasks []clickup.Task `json:"tasks"`
	}
	path := fmt.Sprintf("list/%s/task%s", listID, qs)
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

// CreateTaskLocal creates a task in a list.
func CreateTaskLocal(ctx context.Context, client *api.Client, listID string, req *clickup.TaskRequest, qs string) (*clickup.Task, error) {
	var task clickup.Task
	path := fmt.Sprintf("list/%s/task%s", listID, qs)
	if err := do(ctx, client, "POST", path, req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateTaskLocal updates a task. req is any because both TaskUpdateRequest and
// TaskAssigneeUpdateRequest are used.
func UpdateTaskLocal(ctx context.Context, client *api.Client, taskID string, req any, qs string) (*clickup.Task, error) {
	var task clickup.Task
	path := fmt.Sprintf("task/%s%s", taskID, qs)
	if err := do(ctx, client, "PUT", path, req, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// DeleteTaskLocal deletes a task.
func DeleteTaskLocal(ctx context.Context, client *api.Client, taskID, qs string) error {
	path := fmt.Sprintf("task/%s%s", taskID, qs)
	return do(ctx, client, "DELETE", path, nil, nil)
}

// --- Filtered team tasks ---

// FilteredTeamTasksParams holds parameters for GetFilteredTeamTasksLocal.
type FilteredTeamTasksParams struct {
	OrderBy        string
	Reverse        bool
	Subtasks       bool
	Assignees      []string
	Page           int
	ListIDs        []string
	Statuses       []string
	Tags           []string
	DateUpdGt      int64
	DateUpdLt      int64
	DateCreatedGt  int64
	DateCreatedLt  int64
	DueDateGt      int64
	DueDateLt      int64
	IncludeClosed  bool
	Archived       bool
}

// GetFilteredTeamTasksLocal fetches filtered tasks across a team/workspace.
func GetFilteredTeamTasksLocal(ctx context.Context, client *api.Client, teamID string, params FilteredTeamTasksParams) ([]clickup.Task, error) {
	q := url.Values{}
	if params.OrderBy != "" {
		q.Set("order_by", params.OrderBy)
	}
	if params.Reverse {
		q.Set("reverse", "true")
	}
	if params.Subtasks {
		q.Set("subtasks", "true")
	}
	for _, a := range params.Assignees {
		q.Add("assignees[]", a)
	}
	if params.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", params.Page))
	}
	for _, id := range params.ListIDs {
		q.Add("list_ids[]", id)
	}
	for _, s := range params.Statuses {
		q.Add("statuses[]", s)
	}
	for _, t := range params.Tags {
		q.Add("tags[]", t)
	}
	if params.DateUpdGt > 0 {
		q.Set("date_updated_gt", fmt.Sprintf("%d", params.DateUpdGt))
	}
	if params.DateUpdLt > 0 {
		q.Set("date_updated_lt", fmt.Sprintf("%d", params.DateUpdLt))
	}
	if params.DateCreatedGt > 0 {
		q.Set("date_created_gt", fmt.Sprintf("%d", params.DateCreatedGt))
	}
	if params.DateCreatedLt > 0 {
		q.Set("date_created_lt", fmt.Sprintf("%d", params.DateCreatedLt))
	}
	if params.DueDateGt > 0 {
		q.Set("due_date_gt", fmt.Sprintf("%d", params.DueDateGt))
	}
	if params.DueDateLt > 0 {
		q.Set("due_date_lt", fmt.Sprintf("%d", params.DueDateLt))
	}
	if params.IncludeClosed {
		q.Set("include_closed", "true")
	}
	if params.Archived {
		q.Set("archived", "true")
	}

	path := fmt.Sprintf("team/%s/task?%s", teamID, q.Encode())
	var resp struct {
		Tasks []clickup.Task `json:"tasks"`
	}
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

// --- Hierarchy ---

// GetTeamsLocal fetches all teams (workspaces).
func GetTeamsLocal(ctx context.Context, client *api.Client) ([]clickup.Team, error) {
	var resp struct {
		Teams []clickup.Team `json:"teams"`
	}
	if err := do(ctx, client, "GET", "team", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Teams, nil
}

// GetSpacesLocal fetches spaces for a team.
func GetSpacesLocal(ctx context.Context, client *api.Client, teamID string, archived bool) ([]clickup.Space, error) {
	var resp struct {
		Spaces []clickup.Space `json:"spaces"`
	}
	path := fmt.Sprintf("team/%s/space?archived=%v", teamID, archived)
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Spaces, nil
}

// GetFoldersLocal fetches folders for a space.
func GetFoldersLocal(ctx context.Context, client *api.Client, spaceID string, archived bool) ([]clickup.Folder, error) {
	var resp struct {
		Folders []clickup.Folder `json:"folders"`
	}
	path := fmt.Sprintf("space/%s/folder?archived=%v", spaceID, archived)
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Folders, nil
}

// GetListsLocal fetches lists for a folder.
func GetListsLocal(ctx context.Context, client *api.Client, folderID string, archived bool) ([]clickup.List, error) {
	var resp struct {
		Lists []clickup.List `json:"lists"`
	}
	path := fmt.Sprintf("folder/%s/list?archived=%v", folderID, archived)
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Lists, nil
}

// GetFolderlessListsLocal fetches folderless lists for a space.
func GetFolderlessListsLocal(ctx context.Context, client *api.Client, spaceID string, archived bool) ([]clickup.List, error) {
	var resp struct {
		Lists []clickup.List `json:"lists"`
	}
	path := fmt.Sprintf("space/%s/list?archived=%v", spaceID, archived)
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Lists, nil
}

// GetListLocal fetches a single list.
func GetListLocal(ctx context.Context, client *api.Client, listID string) (*clickup.List, error) {
	var list clickup.List
	path := fmt.Sprintf("list/%s", listID)
	if err := do(ctx, client, "GET", path, nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// --- Custom fields ---

// GetAccessibleCustomFieldsLocal fetches accessible custom fields for a list.
func GetAccessibleCustomFieldsLocal(ctx context.Context, client *api.Client, listID string) ([]clickup.CustomField, error) {
	var resp struct {
		Fields []clickup.CustomField `json:"fields"`
	}
	path := fmt.Sprintf("list/%s/field", listID)
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Fields, nil
}

// SetCustomFieldValueLocal sets a custom field value on a task.
func SetCustomFieldValueLocal(ctx context.Context, client *api.Client, taskID, fieldID string, value any, qs string) error {
	body := map[string]any{"value": value}
	path := fmt.Sprintf("task/%s/field/%s%s", taskID, fieldID, qs)
	return do(ctx, client, "POST", path, body, nil)
}

// RemoveCustomFieldValueLocal removes a custom field value from a task.
func RemoveCustomFieldValueLocal(ctx context.Context, client *api.Client, taskID, fieldID, qs string) error {
	path := fmt.Sprintf("task/%s/field/%s%s", taskID, fieldID, qs)
	return do(ctx, client, "DELETE", path, nil, nil)
}

// --- Dependencies ---

// AddDependencyLocal adds a dependency to a task.
func AddDependencyLocal(ctx context.Context, client *api.Client, taskID string, body any, qs string) error {
	path := fmt.Sprintf("task/%s/dependency%s", taskID, qs)
	return do(ctx, client, "POST", path, body, nil)
}

// DeleteDependencyLocal deletes a dependency from a task.
func DeleteDependencyLocal(ctx context.Context, client *api.Client, taskID, qs string) error {
	path := fmt.Sprintf("task/%s/dependency%s", taskID, qs)
	return do(ctx, client, "DELETE", path, nil, nil)
}

// --- Utility ---

// FetchTeamTasks fetches one page of tasks from the team endpoint with optional
// extra query params. Used by inbox and view for paginated team task fetching.
func FetchTeamTasks(ctx context.Context, client *api.Client, teamID string, page int, extraParams string) ([]clickup.Task, error) {
	path := fmt.Sprintf("team/%s/task?include_closed=true&page=%d&order_by=updated&reverse=true",
		teamID, page)
	if extraParams != "" {
		path += "&" + extraParams
	}
	var resp struct {
		Tasks []clickup.Task `json:"tasks"`
	}
	if err := do(ctx, client, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

// --- User ---

// UserInfo holds the current user's identity from the /user endpoint.
type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// GetUserLocal fetches the currently authenticated user.
func GetUserLocal(ctx context.Context, client *api.Client) (*UserInfo, error) {
	var resp struct {
		User UserInfo `json:"user"`
	}
	if err := do(ctx, client, "GET", "user", nil, &resp); err != nil {
		return nil, err
	}
	return &resp.User, nil
}

// Do is a public wrapper around the unexported do() helper, for use by the
// attachments module.
func Do(ctx context.Context, client *api.Client, method, path string, body any, result any) error {
	return do(ctx, client, method, path, body, result)
}

