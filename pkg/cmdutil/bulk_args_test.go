package cmdutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTaskIDArgs(t *testing.T) {
	tests := []struct {
		name    string
		ids     []string
		wantErr string
	}{
		{"valid native IDs", []string{"86abc1", "86abc2"}, ""},
		{"valid CU- and custom", []string{"CU-abc123", "PROJ-42"}, ""},
		{"empty ID", []string{"a", "", "b"}, "empty task ID"},
		{"id with embedded space", []string{"a b"}, "invalid task ID"},
		{"id with slash", []string{"a/b"}, "invalid task ID"},
		{"id with query marker", []string{"a?b"}, "invalid task ID"},
		{"id with ampersand", []string{"a&b"}, "invalid task ID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTaskIDArgs(tt.ids)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestExpandIDArgs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "already split",
			in:   []string{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			// The zsh case: unquoted variable expansion produces a single
			// arg with embedded spaces. Must split.
			name: "single arg with spaces",
			in:   []string{"a b c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "newline-separated (xargs / stdin pattern)",
			in:   []string{"a\nb\nc"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "mixed split + grouped",
			in:   []string{"a", "b c", "d"},
			want: []string{"a", "b", "c", "d"},
		},
		{
			name: "trailing whitespace dropped",
			in:   []string{"a  b  ", "c\t"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "tabs and multiple spaces collapse",
			in:   []string{"a\t\tb   c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "empty input",
			in:   nil,
			want: []string{},
		},
		{
			name: "all whitespace produces empty",
			in:   []string{"   ", "\t\n"},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandIDArgs(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}
