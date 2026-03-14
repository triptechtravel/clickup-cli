package cmdutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/triptechtravel/clickup-cli/internal/config"
)

func TestCustomIDTaskOptions(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.Config
		isCustomID   bool
		wantNil      bool
		wantCustom   bool
		wantTeamID   int
	}{
		{
			name:       "non-custom ID returns nil",
			cfg:        &config.Config{Workspace: "1276003"},
			isCustomID: false,
			wantNil:    true,
		},
		{
			name:       "custom ID with workspace sets team_id",
			cfg:        &config.Config{Workspace: "1276003"},
			isCustomID: true,
			wantCustom: true,
			wantTeamID: 1276003,
		},
		{
			name:       "custom ID with empty workspace omits team_id",
			cfg:        &config.Config{Workspace: ""},
			isCustomID: true,
			wantCustom: true,
			wantTeamID: 0,
		},
		{
			name:       "custom ID with nil config omits team_id",
			cfg:        nil,
			isCustomID: true,
			wantCustom: true,
			wantTeamID: 0,
		},
		{
			name:       "custom ID with non-numeric workspace omits team_id",
			cfg:        &config.Config{Workspace: "not-a-number"},
			isCustomID: true,
			wantCustom: true,
			wantTeamID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := CustomIDTaskOptions(tt.cfg, tt.isCustomID)
			if tt.wantNil {
				assert.Nil(t, opts)
				return
			}
			assert.NotNil(t, opts)
			assert.Equal(t, tt.wantCustom, opts.CustomTaskIDs)
			assert.Equal(t, tt.wantTeamID, opts.TeamID)
		})
	}
}

func TestCustomIDTaskOptionsWithSubtasks(t *testing.T) {
	cfg := &config.Config{Workspace: "1276003"}

	opts := CustomIDTaskOptionsWithSubtasks(cfg, true)
	assert.True(t, opts.CustomTaskIDs)
	assert.True(t, opts.IncludeSubTasks)
	assert.Equal(t, 1276003, opts.TeamID)

	opts = CustomIDTaskOptionsWithSubtasks(cfg, false)
	assert.False(t, opts.CustomTaskIDs)
	assert.True(t, opts.IncludeSubTasks)
	assert.Equal(t, 0, opts.TeamID)
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
