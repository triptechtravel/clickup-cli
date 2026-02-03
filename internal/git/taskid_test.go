package git

import (
	"testing"
)

func TestExtractTaskID(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		wantID   string
		wantNil  bool
		wantCustom bool
	}{
		{
			name:   "CU hex ID",
			branch: "CU-ae27de-fix-bug",
			wantID: "CU-ae27de",
		},
		{
			name:   "CU hex ID with feature prefix",
			branch: "feature/CU-abc123-new-feature",
			wantID: "CU-abc123",
		},
		{
			name:       "custom prefix ID",
			branch:     "PROJ-42-add-login",
			wantID:     "PROJ-42",
			wantCustom: true,
		},
		{
			name:       "custom prefix with git prefix",
			branch:     "fix/ENG-1234-fix-auth",
			wantID:     "ENG-1234",
			wantCustom: true,
		},
		{
			name:    "no task ID",
			branch:  "my-feature-branch",
			wantNil: true,
		},
		{
			name:    "excluded prefix FEATURE",
			branch:  "FEATURE-123-something",
			wantNil: true,
		},
		{
			name:    "excluded prefix BUGFIX",
			branch:  "BUGFIX-456-something",
			wantNil: true,
		},
		{
			name:   "CU ID case insensitive",
			branch: "cu-DEAD01-test",
			wantID: "cu-DEAD01",
		},
		{
			name:       "multiple segments picks first CU",
			branch:     "CU-aaa111-also-CU-bbb222",
			wantID:     "CU-aaa111",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTaskID(tt.branch)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("expected result, got nil")
			}

			if result.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", result.ID, tt.wantID)
			}

			if result.IsCustomID != tt.wantCustom {
				t.Errorf("IsCustomID = %v, want %v", result.IsCustomID, tt.wantCustom)
			}
		})
	}
}

func TestBranchNamingSuggestion(t *testing.T) {
	suggestion := BranchNamingSuggestion("my-branch")
	if suggestion == "" {
		t.Error("expected non-empty suggestion")
	}
}
