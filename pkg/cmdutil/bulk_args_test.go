package cmdutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
