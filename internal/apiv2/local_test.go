package apiv2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/clickup"
)

func localTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server, api.NewTestClient(server.URL)
}

// ---------------------------------------------------------------------------
// Query helpers
// ---------------------------------------------------------------------------

func TestTaskQuery(t *testing.T) {
	tests := []struct {
		name            string
		customTaskIDs   bool
		teamID          string
		includeSubtasks bool
		want            string
	}{
		{
			name: "no options",
			want: "",
		},
		{
			name:          "custom task IDs with team ID",
			customTaskIDs: true,
			teamID:        "team123",
			want:          "?custom_task_ids=true&team_id=team123",
		},
		{
			name:          "custom task IDs without team ID",
			customTaskIDs: true,
			want:          "?custom_task_ids=true",
		},
		{
			name:            "subtasks only",
			includeSubtasks: true,
			want:            "?include_subtasks=true",
		},
		{
			name:            "all options",
			customTaskIDs:   true,
			teamID:          "t1",
			includeSubtasks: true,
			want:            "?custom_task_ids=true&include_subtasks=true&team_id=t1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TaskQuery(tt.customTaskIDs, tt.teamID, tt.includeSubtasks)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTaskQueryMD(t *testing.T) {
	tests := []struct {
		name          string
		customTaskIDs bool
		teamID        string
		wantContains  []string
	}{
		{
			name:         "no custom IDs",
			wantContains: []string{"include_markdown_description=true", "include_subtasks=true"},
		},
		{
			name:          "with custom IDs and team",
			customTaskIDs: true,
			teamID:        "team1",
			wantContains:  []string{"include_markdown_description=true", "include_subtasks=true", "custom_task_ids=true", "team_id=team1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TaskQueryMD(tt.customTaskIDs, tt.teamID)
			assert.True(t, len(got) > 0 && got[0] == '?', "should start with ?")
			for _, s := range tt.wantContains {
				assert.Contains(t, got, s)
			}
		})
	}
}

func TestDependencyQuery(t *testing.T) {
	tests := []struct {
		name          string
		dependsOn     string
		dependencyOf  string
		customTaskIDs bool
		teamID        string
		want          string
	}{
		{
			name: "empty",
			want: "",
		},
		{
			name:      "depends_on only",
			dependsOn: "abc",
			want:      "?depends_on=abc",
		},
		{
			name:         "dependency_of only",
			dependencyOf: "def",
			want:         "?dependency_of=def",
		},
		{
			name:          "custom task IDs and team",
			customTaskIDs: true,
			teamID:        "t1",
			want:          "?custom_task_ids=true&team_id=t1",
		},
		{
			name:          "all options",
			dependsOn:     "a",
			dependencyOf:  "b",
			customTaskIDs: true,
			teamID:        "t1",
			want:          "?custom_task_ids=true&dependency_of=b&depends_on=a&team_id=t1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DependencyQuery(tt.dependsOn, tt.dependencyOf, tt.customTaskIDs, tt.teamID)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// API wrappers
// ---------------------------------------------------------------------------

const taskJSON = `{
	"id": "task123",
	"custom_id": "PROJ-42",
	"name": "Implement feature",
	"description": "A test task",
	"markdown_description": "# A test task",
	"status": {"status": "in progress", "color": "#4194f6", "type": "custom"},
	"date_created": "1609459200000",
	"date_updated": "1609545600000",
	"creator": {"id": 1, "username": "testuser"},
	"assignees": [{"id": 2, "username": "dev1"}],
	"tags": [{"name": "bug", "tag_fg": "#fff", "tag_bg": "#f00"}],
	"due_date": "1610000000000",
	"time_estimate": 7200000,
	"url": "https://app.clickup.com/t/task123",
	"custom_fields": [{"id":"cf1","name":"Priority Score","type":"number","value":42}]
}`

func TestGetTaskLocal(t *testing.T) {
	var capturedPath string
	var capturedMethod string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(taskJSON))
	})

	task, err := GetTaskLocal(context.Background(), client, "task123", "?custom_task_ids=true&team_id=t1")

	require.NoError(t, err)
	assert.Equal(t, "GET", capturedMethod)
	assert.Contains(t, capturedPath, "/task/task123")

	// Verify decoded fields
	assert.Equal(t, "task123", task.ID)
	assert.Equal(t, "Implement feature", task.Name)
	assert.Equal(t, "in progress", task.Status.Status)
	assert.Len(t, task.Assignees, 1)
	assert.Equal(t, "dev1", task.Assignees[0].Username)
	assert.Len(t, task.Tags, 1)
	assert.Equal(t, "bug", task.Tags[0].Name)
	assert.NotNil(t, task.DueDate)
	assert.Equal(t, "https://app.clickup.com/t/task123", task.URL)
}

func TestGetTasksLocal(t *testing.T) {
	var capturedPath string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tasks":[` + taskJSON + `]}`))
	})

	tasks, err := GetTasksLocal(context.Background(), client, "list1", "?include_subtasks=true")

	require.NoError(t, err)
	assert.Contains(t, capturedPath, "/list/list1/task")
	require.Len(t, tasks, 1)
	assert.Equal(t, "task123", tasks[0].ID)
	assert.Equal(t, "Implement feature", tasks[0].Name)
}

func TestCreateTaskLocal(t *testing.T) {
	var capturedMethod string
	var capturedBody map[string]any

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(taskJSON))
	})

	req := &clickup.TaskRequest{
		Name:        "Implement feature",
		Description: "A test task",
		Status:      "in progress",
	}
	task, err := CreateTaskLocal(context.Background(), client, "list1", req, "")

	require.NoError(t, err)
	assert.Equal(t, "POST", capturedMethod)
	assert.Equal(t, "Implement feature", capturedBody["name"])
	assert.Equal(t, "A test task", capturedBody["description"])
	assert.Equal(t, "in progress", capturedBody["status"])
	assert.Equal(t, "task123", task.ID)
}

func TestUpdateTaskLocal(t *testing.T) {
	var capturedMethod string
	var capturedBody map[string]any
	var capturedPath string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(taskJSON))
	})

	updateReq := map[string]any{"name": "Updated Name", "priority": 2}
	task, err := UpdateTaskLocal(context.Background(), client, "task123", updateReq, "")

	require.NoError(t, err)
	assert.Equal(t, "PUT", capturedMethod)
	assert.Contains(t, capturedPath, "/task/task123")
	assert.Equal(t, "Updated Name", capturedBody["name"])
	assert.EqualValues(t, 2, capturedBody["priority"])
	assert.Equal(t, "task123", task.ID)
}

func TestDeleteTaskLocal(t *testing.T) {
	var capturedMethod string
	var capturedPath string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	err := DeleteTaskLocal(context.Background(), client, "task123", "?custom_task_ids=true&team_id=t1")

	require.NoError(t, err)
	assert.Equal(t, "DELETE", capturedMethod)
	assert.Contains(t, capturedPath, "/task/task123")
}

func TestGetFilteredTeamTasksLocal(t *testing.T) {
	var capturedPath string
	var capturedQuery string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tasks":[` + taskJSON + `]}`))
	})

	params := FilteredTeamTasksParams{
		OrderBy:       "due_date",
		Reverse:       true,
		Subtasks:      true,
		Assignees:     []string{"user1", "user2"},
		Page:          2,
		ListIDs:       []string{"list1"},
		Statuses:      []string{"open", "in progress"},
		Tags:          []string{"bug"},
		DateUpdGt:     1700000000000,
		DateUpdLt:     1700100000000,
		IncludeClosed: true,
	}

	tasks, err := GetFilteredTeamTasksLocal(context.Background(), client, "team1", params)

	require.NoError(t, err)
	assert.Contains(t, capturedPath, "/team/team1/task")
	require.Len(t, tasks, 1)

	// Verify all query params are encoded
	assert.Contains(t, capturedQuery, "order_by=due_date")
	assert.Contains(t, capturedQuery, "reverse=true")
	assert.Contains(t, capturedQuery, "subtasks=true")
	assert.Contains(t, capturedQuery, "assignees%5B%5D=user1")
	assert.Contains(t, capturedQuery, "assignees%5B%5D=user2")
	assert.Contains(t, capturedQuery, "page=2")
	assert.Contains(t, capturedQuery, "list_ids%5B%5D=list1")
	assert.Contains(t, capturedQuery, "statuses%5B%5D=open")
	assert.Contains(t, capturedQuery, "statuses%5B%5D=in+progress")
	assert.Contains(t, capturedQuery, "tags%5B%5D=bug")
	assert.Contains(t, capturedQuery, "date_updated_gt=1700000000000")
	assert.Contains(t, capturedQuery, "date_updated_lt=1700100000000")
	assert.Contains(t, capturedQuery, "include_closed=true")
}

func TestGetFilteredTeamTasksLocal_EmptyParams(t *testing.T) {
	var capturedQuery string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tasks":[]}`))
	})

	tasks, err := GetFilteredTeamTasksLocal(context.Background(), client, "team1", FilteredTeamTasksParams{})

	require.NoError(t, err)
	assert.Empty(t, tasks)
	// With no params set, query string should be empty (or nearly so)
	assert.NotContains(t, capturedQuery, "order_by")
	assert.NotContains(t, capturedQuery, "reverse")
	assert.NotContains(t, capturedQuery, "subtasks")
}

func TestGetTeamsLocal(t *testing.T) {
	var capturedPath string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"teams":[{"id":"t1","name":"My Workspace","color":"#000"},{"id":"t2","name":"Other","color":"#fff"}]}`))
	})

	teams, err := GetTeamsLocal(context.Background(), client)

	require.NoError(t, err)
	assert.Equal(t, "/api/v2/team", capturedPath)
	require.Len(t, teams, 2)
	assert.Equal(t, "t1", teams[0].ID)
	assert.Equal(t, "My Workspace", teams[0].Name)
	assert.Equal(t, "t2", teams[1].ID)
}

func TestGetFoldersLocal(t *testing.T) {
	tests := []struct {
		name     string
		archived bool
		wantQS   string
	}{
		{"not archived", false, "archived=false"},
		{"archived", true, "archived=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			var capturedQuery string

			_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				capturedQuery = r.URL.RawQuery
				w.Header().Set("X-RateLimit-Remaining", "99")
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"folders":[{"id":"f1","name":"Sprint Folder","orderindex":"0"}]}`))
			})

			folders, err := GetFoldersLocal(context.Background(), client, "space1", tt.archived)

			require.NoError(t, err)
			assert.Contains(t, capturedPath, "/space/space1/folder")
			assert.Contains(t, capturedQuery, tt.wantQS)
			require.Len(t, folders, 1)
			assert.Equal(t, "f1", folders[0].ID)
			assert.Equal(t, "Sprint Folder", folders[0].Name)
		})
	}
}

func TestGetListsLocal(t *testing.T) {
	tests := []struct {
		name     string
		archived bool
		wantQS   string
	}{
		{"not archived", false, "archived=false"},
		{"archived", true, "archived=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			var capturedQuery string

			_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				capturedQuery = r.URL.RawQuery
				w.Header().Set("X-RateLimit-Remaining", "99")
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"lists":[{"id":"l1","name":"Backlog","orderindex":"1"},{"id":"l2","name":"Sprint 1","orderindex":"2"}]}`))
			})

			lists, err := GetListsLocal(context.Background(), client, "folder1", tt.archived)

			require.NoError(t, err)
			assert.Contains(t, capturedPath, "/folder/folder1/list")
			assert.Contains(t, capturedQuery, tt.wantQS)
			require.Len(t, lists, 2)
			assert.Equal(t, "l1", lists[0].ID)
			assert.Equal(t, "Backlog", lists[0].Name)
			assert.Equal(t, "l2", lists[1].ID)
		})
	}
}

func TestGetListLocal(t *testing.T) {
	var capturedPath string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		// NOT wrapped in {"lists":[...]} -- single list response
		w.Write([]byte(`{"id":"l1","name":"My List","orderindex":"0","content":"List description"}`))
	})

	list, err := GetListLocal(context.Background(), client, "l1")

	require.NoError(t, err)
	assert.Equal(t, "/api/v2/list/l1", capturedPath)
	assert.Equal(t, "l1", list.ID)
	assert.Equal(t, "My List", list.Name)
	assert.Equal(t, "List description", list.Content)
}

func TestGetAccessibleCustomFieldsLocal(t *testing.T) {
	var capturedPath string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"fields":[{"id":"cf1","name":"Story Points","type":"number"},{"id":"cf2","name":"Sprint","type":"drop_down"}]}`))
	})

	fields, err := GetAccessibleCustomFieldsLocal(context.Background(), client, "list1")

	require.NoError(t, err)
	assert.Equal(t, "/api/v2/list/list1/field", capturedPath)
	require.Len(t, fields, 2)
	assert.Equal(t, "cf1", fields[0].ID)
	assert.Equal(t, "Story Points", fields[0].Name)
	assert.Equal(t, "number", fields[0].Type)
	assert.Equal(t, "cf2", fields[1].ID)
	assert.Equal(t, "drop_down", fields[1].Type)
}

func TestSetCustomFieldValueLocal(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedBody map[string]any

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	err := SetCustomFieldValueLocal(context.Background(), client, "task1", "field1", 42, "?custom_task_ids=true")

	require.NoError(t, err)
	assert.Equal(t, "POST", capturedMethod)
	assert.Contains(t, capturedPath, "/task/task1/field/field1")
	// The body should wrap the value: {"value": 42}
	assert.EqualValues(t, 42, capturedBody["value"])
}

func TestSetCustomFieldValueLocal_StringValue(t *testing.T) {
	var capturedBody map[string]any

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	err := SetCustomFieldValueLocal(context.Background(), client, "task1", "field1", "hello", "")
	require.NoError(t, err)
	assert.Equal(t, "hello", capturedBody["value"])
}

func TestAddDependencyLocal(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedBody map[string]any

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	depBody := map[string]string{"depends_on": "other_task"}
	err := AddDependencyLocal(context.Background(), client, "task1", depBody, "?custom_task_ids=true&team_id=t1")

	require.NoError(t, err)
	assert.Equal(t, "POST", capturedMethod)
	assert.Contains(t, capturedPath, "/task/task1/dependency")
	assert.Equal(t, "other_task", capturedBody["depends_on"])
}

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

func TestGetTaskLocal_APIError(t *testing.T) {
	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"err":"Task not found","ECODE":"ITEM_015"}`))
	})

	task, err := GetTaskLocal(context.Background(), client, "nonexistent", "")

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "404")
}

func TestCreateTaskLocal_APIError(t *testing.T) {
	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"err":"List not found","ECODE":"LIST_001"}`))
	})

	task, err := CreateTaskLocal(context.Background(), client, "bad_list", &clickup.TaskRequest{Name: "fail"}, "")

	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "400")
}

// ---------------------------------------------------------------------------
// Additional wrappers not in the explicit list but present in local.go
// ---------------------------------------------------------------------------

func TestGetSpacesLocal(t *testing.T) {
	var capturedPath string
	var capturedQuery string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"spaces":[{"id":"s1","name":"Engineering","private":false}]}`))
	})

	spaces, err := GetSpacesLocal(context.Background(), client, "team1", false)

	require.NoError(t, err)
	assert.Contains(t, capturedPath, "/team/team1/space")
	assert.Contains(t, capturedQuery, "archived=false")
	require.Len(t, spaces, 1)
	assert.Equal(t, "s1", spaces[0].ID)
	assert.Equal(t, "Engineering", spaces[0].Name)
}

func TestGetFolderlessListsLocal(t *testing.T) {
	var capturedPath string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"lists":[{"id":"l5","name":"Standalone List","orderindex":"0"}]}`))
	})

	lists, err := GetFolderlessListsLocal(context.Background(), client, "space1", false)

	require.NoError(t, err)
	assert.Equal(t, "/api/v2/space/space1/list", capturedPath)
	require.Len(t, lists, 1)
	assert.Equal(t, "l5", lists[0].ID)
}

func TestRemoveCustomFieldValueLocal(t *testing.T) {
	var capturedMethod string
	var capturedPath string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	err := RemoveCustomFieldValueLocal(context.Background(), client, "task1", "field1", "?custom_task_ids=true")

	require.NoError(t, err)
	assert.Equal(t, "DELETE", capturedMethod)
	assert.Contains(t, capturedPath, "/task/task1/field/field1")
}

func TestDeleteDependencyLocal(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedQuery string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	err := DeleteDependencyLocal(context.Background(), client, "task1", "?depends_on=other&custom_task_ids=true&team_id=t1")

	require.NoError(t, err)
	assert.Equal(t, "DELETE", capturedMethod)
	assert.Contains(t, capturedPath, "/task/task1/dependency")
	assert.Contains(t, capturedQuery, "depends_on=other")
	assert.Contains(t, capturedQuery, "custom_task_ids=true")
}

func TestFetchTeamTasks(t *testing.T) {
	var capturedPath string
	var capturedQuery string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tasks":[` + taskJSON + `]}`))
	})

	tasks, err := FetchTeamTasks(context.Background(), client, "team1", 3, "assignees%5B%5D=user1")

	require.NoError(t, err)
	assert.Contains(t, capturedPath, "/team/team1/task")
	assert.Contains(t, capturedQuery, "include_closed=true")
	assert.Contains(t, capturedQuery, "page=3")
	assert.Contains(t, capturedQuery, "order_by=updated")
	assert.Contains(t, capturedQuery, "reverse=true")
	assert.Contains(t, capturedQuery, "assignees%5B%5D=user1")
	require.Len(t, tasks, 1)
	assert.Equal(t, "task123", tasks[0].ID)
}

func TestFetchTeamTasks_NoExtraParams(t *testing.T) {
	var capturedQuery string

	_, client := localTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tasks":[]}`))
	})

	tasks, err := FetchTeamTasks(context.Background(), client, "team1", 0, "")

	require.NoError(t, err)
	assert.Empty(t, tasks)
	assert.Contains(t, capturedQuery, "page=0")
	assert.NotContains(t, capturedQuery, "&&")
}
