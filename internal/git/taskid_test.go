package git

import (
	"testing"
)

func TestExtractTaskID(t *testing.T) {
	tests := []struct {
		name       string
		branch     string
		wantID     string
		wantRaw    string
		wantNil    bool
		wantCustom bool
	}{
		{
			name:   "CU hex ID",
			branch: "CU-ae27de-fix-bug",
			wantID: "ae27de",
			wantRaw: "CU-ae27de",
		},
		{
			name:   "CU hex ID with feature prefix",
			branch: "feature/CU-abc123-new-feature",
			wantID: "abc123",
			wantRaw: "CU-abc123",
		},
		{
			name:   "CU alphanumeric ID",
			branch: "CU-86d1u2bz4_React-Native-Pois-gone",
			wantID: "86d1u2bz4",
			wantRaw: "CU-86d1u2bz4",
		},
		{
			name:   "CU alphanumeric ID with feature prefix",
			branch: "features/CU-86d0xd2r1_BUG-React-Native",
			wantID: "86d0xd2r1",
			wantRaw: "CU-86d0xd2r1",
		},
		{
			name:       "custom prefix ID",
			branch:     "PROJ-42-add-login",
			wantID:     "PROJ-42",
			wantRaw:    "PROJ-42",
			wantCustom: true,
		},
		{
			name:       "custom prefix with git prefix",
			branch:     "fix/ENG-1234-fix-auth",
			wantID:     "ENG-1234",
			wantRaw:    "ENG-1234",
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
			branch: "cu-dead01-test",
			wantID: "dead01",
			wantRaw: "cu-dead01",
		},
		{
			name:   "multiple segments picks first CU",
			branch: "CU-aaa111-also-CU-bbb222",
			wantID: "aaa111",
			wantRaw: "CU-aaa111",
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

			if tt.wantRaw != "" && result.Raw != tt.wantRaw {
				t.Errorf("Raw = %q, want %q", result.Raw, tt.wantRaw)
			}

			if result.IsCustomID != tt.wantCustom {
				t.Errorf("IsCustomID = %v, want %v", result.IsCustomID, tt.wantCustom)
			}
		})
	}
}

func TestParseTaskID(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantID     string
		wantCustom bool
	}{
		{
			name:   "CU-prefixed ID strips prefix",
			input:  "CU-86d1u2bz4",
			wantID: "86d1u2bz4",
		},
		{
			name:   "CU-prefixed hex ID strips prefix",
			input:  "CU-abc123",
			wantID: "abc123",
		},
		{
			name:   "CU-prefixed case insensitive",
			input:  "cu-dead01",
			wantID: "dead01",
		},
		{
			name:   "raw ID passes through",
			input:  "86d1u2bz4",
			wantID: "86d1u2bz4",
		},
		{
			name:       "custom prefix ID",
			input:      "PROJ-42",
			wantID:     "PROJ-42",
			wantCustom: true,
		},
		{
			name:       "custom prefix ID ENG",
			input:      "ENG-1234",
			wantID:     "ENG-1234",
			wantCustom: true,
		},
		{
			name:   "excluded prefix passes through as raw",
			input:  "FEATURE-123",
			wantID: "FEATURE-123",
		},
		{
			name:   "empty string",
			input:  "",
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTaskID(tt.input)

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
