package cmdutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/triptechtravel/clickup-cli/internal/config"
)

func TestCustomIDTaskQuery(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.Config
		isCustomID bool
		wantEmpty  bool
		wantContains []string
	}{
		{
			name:       "non-custom ID returns empty",
			cfg:        &config.Config{Workspace: "1276003"},
			isCustomID: false,
			wantEmpty:  true,
		},
		{
			name:       "custom ID with workspace sets team_id",
			cfg:        &config.Config{Workspace: "1276003"},
			isCustomID: true,
			wantContains: []string{"custom_task_ids=true", "team_id=1276003"},
		},
		{
			name:       "custom ID with empty workspace omits team_id",
			cfg:        &config.Config{Workspace: ""},
			isCustomID: true,
			wantContains: []string{"custom_task_ids=true"},
		},
		{
			name:       "custom ID with nil config omits team_id",
			cfg:        nil,
			isCustomID: true,
			wantContains: []string{"custom_task_ids=true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qs := CustomIDTaskQuery(tt.cfg, tt.isCustomID)
			if tt.wantEmpty {
				assert.Empty(t, qs)
				return
			}
			for _, s := range tt.wantContains {
				assert.Contains(t, qs, s)
			}
		})
	}
}

func TestCustomIDTaskQueryWithSubtasks(t *testing.T) {
	cfg := &config.Config{Workspace: "1276003"}

	qs := CustomIDTaskQueryWithSubtasks(cfg, true)
	assert.Contains(t, qs, "custom_task_ids=true")
	assert.Contains(t, qs, "include_subtasks=true")
	assert.Contains(t, qs, "team_id=1276003")

	qs = CustomIDTaskQueryWithSubtasks(cfg, false)
	assert.NotContains(t, qs, "custom_task_ids=true")
	assert.Contains(t, qs, "include_subtasks=true")
}

func TestCustomIDQueryParam(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.Config
		isCustomID bool
		want       string
	}{
		{
			name:       "non-custom returns empty",
			cfg:        &config.Config{Workspace: "1276003"},
			isCustomID: false,
			want:       "",
		},
		{
			name:       "custom with workspace",
			cfg:        &config.Config{Workspace: "1276003"},
			isCustomID: true,
			want:       "?custom_task_ids=true&team_id=1276003",
		},
		{
			name:       "custom with empty workspace",
			cfg:        &config.Config{Workspace: ""},
			isCustomID: true,
			want:       "?custom_task_ids=true",
		},
		{
			name:       "custom with nil config",
			cfg:        nil,
			isCustomID: true,
			want:       "?custom_task_ids=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CustomIDQueryParam(tt.cfg, tt.isCustomID))
		})
	}
}
