// Package clickup provides local type definitions for ClickUp API v2 entities.
// These types are forked from github.com/raksul/go-clickup/clickup to remove
// that external dependency while preserving identical JSON serialisation.
package clickup

import (
	"encoding/json"
)

// User represents a ClickUp user.
type User struct {
	ID                int    `json:"id"`
	Username          string `json:"username"`
	Email             string `json:"email"`
	Color             string `json:"color"`
	ProfilePicture    string `json:"profilePicture,omitempty"`
	Initials          string `json:"initials"`
	WeekStartDay      int    `json:"week_start_day,omitempty"`
	GlobalFontSupport bool   `json:"global_font_support"`
	Timezone          string `json:"timezone"`
}

// Tag represents a ClickUp tag on a task.
type Tag struct {
	Name    string `json:"name"`
	TagFg   string `json:"tag_fg"`
	TagBg   string `json:"tag_bg"`
	Creator int    `json:"creator,omitempty"`
}

// Task represents a ClickUp task.
type Task struct {
	ID                  string                 `json:"id"`
	CustomID            string                 `json:"custom_id"`
	CustomItemId        int                    `json:"custom_item_id"`
	Name                string                 `json:"name"`
	TextContent         string                 `json:"text_content"`
	Description         string                 `json:"description"`
	MarkdownDescription string                 `json:"markdown_description"`
	Status              TaskStatus             `json:"status"`
	Orderindex          json.Number            `json:"orderindex"`
	DateCreated         string                 `json:"date_created"`
	DateUpdated         string                 `json:"date_updated"`
	DateClosed          string                 `json:"date_closed"`
	Archived            bool                   `json:"archived"`
	Creator             User                   `json:"creator"`
	Assignees           []User                 `json:"assignees,omitempty"`
	Watchers            []User                 `json:"watchers,omitempty"`
	Checklists          []Checklist            `json:"checklists,omitempty"`
	Tags                []Tag                  `json:"tags,omitempty"`
	Parent              string                 `json:"parent"`
	Priority            TaskPriority           `json:"priority"`
	DueDate             *Date                  `json:"due_date,omitempty"`
	StartDate           string                 `json:"start_date,omitempty"`
	Points              Point                  `json:"points,omitempty"`
	TimeEstimate        int64                  `json:"time_estimate"`
	TimeSpent           int64                  `json:"time_spent"`
	CustomFields        []CustomField          `json:"custom_fields"`
	Dependencies        []Dependence           `json:"dependencies"`
	LinkedTasks         []LinkedTask           `json:"linked_tasks"`
	TeamID              string                 `json:"team_id"`
	URL                 string                 `json:"url"`
	PermissionLevel     string                 `json:"permission_level"`
	List                ListOfTaskBelonging    `json:"list"`
	Project             ProjectOfTaskBelonging `json:"project"`
	Folder              FolderOftaskBelonging  `json:"folder"`
	Space               SpaceOfTaskBelonging   `json:"space"`
	Attachments         []TaskAttachment       `json:"attachments"`
}

// TaskStatus represents a task's status.
type TaskStatus struct {
	ID         string      `json:"id"`
	Status     string      `json:"status"`
	Color      string      `json:"color"`
	Type       string      `json:"type"`
	Orderindex json.Number `json:"orderindex"`
}

// TaskPriority represents a task's priority.
type TaskPriority struct {
	Priority string `json:"priority"`
	Color    string `json:"color"`
}

// TaskAttachment represents an attachment on a task.
type TaskAttachment struct {
	ID               string `json:"id"`
	Date             string `json:"date"`
	Title            string `json:"title"`
	Type             int    `json:"type"`
	Source           int    `json:"source"`
	Version          int    `json:"version"`
	Extension        string `json:"extension"`
	ThumbnailSmall   string `json:"thumbnail_small"`
	ThumbnailMedium  string `json:"thumbnail_medium"`
	ThumbnailLarge   string `json:"thumbnail_large"`
	IsFolder         bool   `json:"is_folder"`
	Mimetype         string `json:"mimetype"`
	Hidden           bool   `json:"hidden"`
	ParentId         string `json:"parent_id"`
	Size             int    `json:"size"`
	TotalComments    int    `json:"total_comments"`
	ResolvedComments int    `json:"resolved_comments"`
	User             User   `json:"user"`
	Deleted          bool   `json:"deleted"`
	Orientation      string `json:"orientation"`
	Url              string `json:"url"`
	EmailData        string `json:"email_data"`
	UrlWQuery        string `json:"url_w_query"`
	UrlWHost         string `json:"url_w_host"`
}

// Attachment represents a file attachment for upload.
type Attachment struct {
	FileName string
	Reader   interface{ Read([]byte) (int, error) }
}

// TaskAttachementOptions holds options for task attachment operations.
// Note: the typo "Attachement" is preserved from go-clickup for compatibility.
type TaskAttachementOptions struct {
	CustomTaskIDs bool `url:"custom_task_ids,omitempty"`
	TeamID        int  `url:"team_id,omitempty"`
}

// Dependence represents a dependency relationship between tasks.
type Dependence struct {
	TaskID      string `json:"task_id"`
	DependsOn   string `json:"depends_on"`
	Type        int    `json:"type"`
	DateCreated string `json:"date_created"`
	Userid      string `json:"userid"`
}

// LinkedTask represents a link between tasks.
type LinkedTask struct {
	TaskID      string `json:"task_id"`
	LinkID      string `json:"link_id"`
	DateCreated string `json:"date_created"`
	Userid      string `json:"userid"`
}

// ListOfTaskBelonging identifies the list a task belongs to.
type ListOfTaskBelonging struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Access bool   `json:"access"`
}

// ProjectOfTaskBelonging identifies the project a task belongs to.
type ProjectOfTaskBelonging struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hidden bool   `json:"hidden"`
	Access bool   `json:"access"`
}

// FolderOftaskBelonging identifies the folder a task belongs to.
// Note: the typo "Oftask" is preserved from go-clickup for compatibility.
type FolderOftaskBelonging struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hidden bool   `json:"hidden"`
	Access bool   `json:"access"`
}

// SpaceOfTaskBelonging identifies the space a task belongs to.
type SpaceOfTaskBelonging struct {
	ID string `json:"id"`
}

// TaskRequest is used to create a new task.
type TaskRequest struct {
	Name                      string                     `json:"name,omitempty"`
	Description               string                     `json:"description,omitempty"`
	MarkdownDescription       string                     `json:"markdown_description,omitempty"`
	Assignees                 []int                      `json:"assignees,omitempty"`
	Tags                      []string                   `json:"tags,omitempty"`
	Status                    string                     `json:"status,omitempty"`
	Priority                  int                        `json:"priority,omitempty"`
	DueDate                   *Date                      `json:"due_date,omitempty"`
	DueDateTime               bool                       `json:"due_date_time,omitempty"`
	TimeEstimate              int                        `json:"time_estimate,omitempty"`
	StartDate                 *Date                      `json:"start_date,omitempty"`
	StartDateTime             bool                       `json:"start_date_time,omitempty"`
	NotifyAll                 bool                       `json:"notify_all,omitempty"`
	Parent                    string                     `json:"parent,omitempty"`
	LinksTo                   string                     `json:"links_to,omitempty"`
	CheckRequiredCustomFields bool                       `json:"check_required_custom_fields,omitempty"`
	CustomFields              []CustomFieldInTaskRequest `json:"custom_fields,omitempty"`
	CustomItemId              int                        `json:"custom_item_id,omitempty"`
}

// TaskUpdateRequest is used to update an existing task.
type TaskUpdateRequest struct {
	Name                      string                     `json:"name,omitempty"`
	Description               string                     `json:"description,omitempty"`
	Assignees                 TaskAssigneeUpdateRequest  `json:"assignees,omitempty"`
	Tags                      []string                   `json:"tags,omitempty"`
	Status                    string                     `json:"status,omitempty"`
	Priority                  int                        `json:"priority,omitempty"`
	DueDate                   *Date                      `json:"due_date,omitempty"`
	DueDateTime               bool                       `json:"due_date_time,omitempty"`
	TimeEstimate              int                        `json:"time_estimate,omitempty"`
	StartDate                 *Date                      `json:"start_date,omitempty"`
	StartDateTime             bool                       `json:"start_date_time,omitempty"`
	NotifyAll                 bool                       `json:"notify_all,omitempty"`
	Parent                    string                     `json:"parent,omitempty"`
	LinksTo                   string                     `json:"links_to,omitempty"`
	CheckRequiredCustomFields bool                       `json:"check_required_custom_fields,omitempty"`
	CustomFields              []CustomFieldInTaskRequest `json:"custom_fields,omitempty"`
	CustomItemId              int                        `json:"custom_item_id,omitempty"`
}

// TaskAssigneeUpdateRequest is used to add/remove assignees on a task update.
type TaskAssigneeUpdateRequest struct {
	Add []int `json:"add,omitempty"`
	Rem []int `json:"rem,omitempty"`
}

// CustomFieldInTaskRequest is used to set custom fields when creating/updating tasks.
type CustomFieldInTaskRequest struct {
	ID    string      `json:"id"`
	Value interface{} `json:"value"`
}

// GetTaskOptions holds options for a single task GET request.
type GetTaskOptions struct {
	CustomTaskIDs   bool `url:"custom_task_ids,omitempty"`
	TeamID          int  `url:"team_id,omitempty"`
	IncludeSubTasks bool `url:"include_subtasks,omitempty"`
}

// GetTasksOptions holds options for listing tasks.
type GetTasksOptions struct {
	Archived      bool     `url:"archived,omitempty"`
	Page          int      `url:"page,omitempty"`
	OrderBy       string   `url:"order_by,omitempty"`
	Reverse       bool     `url:"reverse,omitempty"`
	Subtasks      bool     `url:"subtasks,omitempty"`
	Statuses      []string `url:"statuses[],omitempty"`
	IncludeClosed bool     `url:"include_closed,omitempty"`
	Assignees     []string `url:"assignees[],omitempty"`
	Tags          []string `url:"tags[],omitempty"`
	DueDateGt     *Date    `url:"due_date_gt,omitempty"`
	DueDateLt     *Date    `url:"due_date_lt,omitempty"`
	DateCreatedGt *Date    `url:"date_created_gt,omitempty"`
	DateCreatedLt *Date    `url:"date_created_lt,omitempty"`
	DateUpdatedGt *Date    `url:"date_updated_gt,omitempty"`
	DateUpdatedLt *Date    `url:"date_updated_lt,omitempty"`
}

// AddDependencyRequest is used to add a dependency to a task.
type AddDependencyRequest struct {
	DependsOn    string `json:"depends_on,omitempty"`
	DependencyOf string `json:"dependency_of,omitempty"`
}

// DeleteDependencyOptions holds options for deleting a dependency.
type DeleteDependencyOptions struct {
	DependsOn     string `url:"depends_on,omitempty"`
	DependencyOf  string `url:"dependency_of,omitempty"`
	CustomTaskIDs string `url:"custom_task_ids,omitempty"`
	TeamID        int    `url:"team_id,omitempty"`
}

// Checklist represents a task checklist.
type Checklist struct {
	ID         string      `json:"id"`
	TaskID     string      `json:"task_id"`
	Name       string      `json:"name"`
	Orderindex json.Number `json:"orderindex"`
	Resolved   int         `json:"resolved"`
	Unresolved int         `json:"unresolved"`
	Items      []Item      `json:"items,omitempty"`
}

// Item represents a checklist item.
type Item struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Orderindex  json.Number   `json:"orderindex"`
	Assignee    User          `json:"assignee"`
	Resolved    bool          `json:"resolved"`
	Parent      interface{}   `json:"parent"`
	DateCreated string        `json:"date_created"`
	Children    []interface{} `json:"children"`
}

// ChecklistRequest is used to create/update a checklist.
type ChecklistRequest struct {
	Name     string `json:"name"`
	Position int    `json:"position,omitempty"`
}

// ChecklistItemRequest is used to create/update a checklist item.
type ChecklistItemRequest struct {
	Name     string `json:"name"`
	Assignee int    `json:"assignee,omitempty"`
	Resolved bool   `json:"resolved,omitempty"`
}

// Team represents a ClickUp workspace (team).
type Team struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Color   string       `json:"color"`
	Avatar  interface{}  `json:"avatar"`
	Members []TeamMember `json:"members"`
}

// TeamMember represents a member of a team.
type TeamMember struct {
	User      TeamUser  `json:"user"`
	InvitedBy InvitedBy `json:"invited_by,omitempty"`
}

// TeamUser represents the user info within a team member.
type TeamUser struct {
	ID             int         `json:"id"`
	Username       string      `json:"username"`
	Email          string      `json:"email"`
	Color          string      `json:"color"`
	ProfilePicture string      `json:"profilePicture"`
	Initials       string      `json:"initials"`
	Role           int         `json:"role"`
	CustomRole     interface{} `json:"custom_role"`
	LastActive     string      `json:"last_active"`
	DateJoined     string      `json:"date_joined"`
	DateInvited    string      `json:"date_invited"`
}

// InvitedBy represents the user who invited a team member.
type InvitedBy struct {
	ID             int    `json:"id"`
	Username       string `json:"username"`
	Color          string `json:"color"`
	Email          string `json:"email"`
	Initials       string `json:"initials"`
	ProfilePicture string `json:"profilePicture"`
}

// Space represents a ClickUp space.
type Space struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Private  bool   `json:"private"`
	Statuses []struct {
		ID         string      `json:"id"`
		Status     string      `json:"status"`
		Type       string      `json:"type"`
		Orderindex json.Number `json:"orderindex"`
		Color      string      `json:"color"`
	} `json:"statuses"`
	MultipleAssignees bool `json:"multiple_assignees"`
	Features          struct {
		DueDates DueDates `json:"due_dates"`
		Sprints  struct {
			Enabled bool `json:"enabled"`
		} `json:"sprints"`
		TimeTracking TimeTracking `json:"time_tracking"`
		Points       struct {
			Enabled bool `json:"enabled"`
		} `json:"points"`
		CustomItems struct {
			Enabled bool `json:"enabled"`
		} `json:"custom_items"`
		Tags            Tags          `json:"tags"`
		TimeEstimates   TimeEstimates `json:"time_estimates"`
		CheckUnresolved struct {
			Enabled    bool        `json:"enabled"`
			Subtasks   bool        `json:"subtasks"`
			Checklists interface{} `json:"checklists"`
			Comments   interface{} `json:"comments"`
		} `json:"check_unresolved"`
		Zoom struct {
			Enabled bool `json:"enabled"`
		} `json:"zoom"`
		Milestones struct {
			Enabled bool `json:"enabled"`
		} `json:"milestones"`
		RemapDependencies RemapDependencies `json:"remap_dependencies"`
		DependencyWarning DependencyWarning `json:"dependency_warning"`
		MultipleAssignees struct {
			Enabled bool `json:"enabled"`
		} `json:"multiple_assignees"`
		Emails struct {
			Enabled bool `json:"enabled"`
		} `json:"emails"`
	} `json:"features"`
	Archived bool `json:"archived"`
}

// DueDates is a space feature toggle for due dates.
type DueDates struct {
	Enabled            bool `json:"enabled"`
	StartDate          bool `json:"start_date"`
	RemapDueDates      bool `json:"remap_due_dates"`
	RemapClosedDueDate bool `json:"remap_closed_due_date"`
}

// TimeTracking is a space feature toggle for time tracking.
type TimeTracking struct {
	Enabled bool `json:"enabled"`
}

// Tags is a space feature toggle for tags.
type Tags struct {
	Enabled bool `json:"enabled"`
}

// TimeEstimates is a space feature toggle for time estimates.
type TimeEstimates struct {
	Enabled     bool `json:"enabled"`
	Rollup      bool `json:"rollup"`
	PerAssignee bool `json:"per_assignee"`
}

// RemapDependencies is a space feature toggle.
type RemapDependencies struct {
	Enabled bool `json:"enabled"`
}

// DependencyWarning is a space feature toggle.
type DependencyWarning struct {
	Enabled bool `json:"enabled"`
}

// NullDate returns a Date that marshals to JSON null.
func NullDate() *Date {
	return &Date{null: true}
}

// Folder represents a ClickUp folder.
type Folder struct {
	ID               string                  `json:"id"`
	Name             string                  `json:"name"`
	Orderindex       json.Number             `json:"orderindex"`
	OverrideStatuses bool                    `json:"override_statuses"`
	Hidden           bool                    `json:"hidden"`
	Space            SpaceOfFolderBelonging  `json:"space"`
	TaskCount        json.Number             `json:"task_count"`
	Archived         bool                    `json:"archived"`
	Statuses         []interface{}           `json:"statuses"`
	Lists            []ListOfFolderBelonging `json:"lists"`
	PermissionLevel  string                  `json:"permission_level"`
}

// SpaceOfFolderBelonging identifies the space a folder belongs to.
type SpaceOfFolderBelonging struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Access bool   `json:"access,omitempty"`
}

// ListOfFolderBelonging identifies a list within a folder.
type ListOfFolderBelonging struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Orderindex json.Number `json:"orderindex"`
	Status     interface{} `json:"status"`
	Priority   interface{} `json:"priority"`
	Assignee   interface{} `json:"assignee"`
	TaskCount  json.Number `json:"task_count"`
	DueDate    interface{} `json:"due_date"`
	StartDate  interface{} `json:"start_date"`
	Space      struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Access bool   `json:"access"`
	} `json:"space"`
	Archived         bool        `json:"archived"`
	OverrideStatuses interface{} `json:"override_statuses"`
	Statuses         []struct {
		ID         string      `json:"id"`
		Status     string      `json:"status"`
		Orderindex json.Number `json:"orderindex"`
		Color      string      `json:"color"`
		Type       string      `json:"type"`
	} `json:"statuses"`
	PermissionLevel string `json:"permission_level,omitempty"`
}

// List represents a ClickUp list.
type List struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Orderindex json.Number `json:"orderindex"`
	Content    string      `json:"content"`
	Status     struct {
		Status    string `json:"status"`
		Color     string `json:"color"`
		HideLabel bool   `json:"hide_label"`
	} `json:"status"`
	Priority struct {
		Priority string `json:"priority"`
		Color    string `json:"color"`
	} `json:"priority"`
	Assignee      User        `json:"assignee,omitempty"`
	TaskCount     json.Number `json:"task_count"`
	DueDate       string      `json:"due_date"`
	DueDateTime   bool        `json:"due_date_time"`
	StartDate     string      `json:"start_date"`
	StartDateTime bool        `json:"start_date_time"`
	Folder        struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Hidden bool   `json:"hidden"`
		Access bool   `json:"access"`
	} `json:"folder"`
	Space struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Access bool   `json:"access"`
	} `json:"space"`
	Statuses []struct {
		Status     string      `json:"status"`
		Orderindex json.Number `json:"orderindex"`
		Color      string      `json:"color"`
		Type       string      `json:"type"`
	} `json:"statuses"`
	InboundAddress  string `json:"inbound_address"`
	Archived        bool   `json:"archived"`
	PermissionLevel string `json:"permission_level"`
}
