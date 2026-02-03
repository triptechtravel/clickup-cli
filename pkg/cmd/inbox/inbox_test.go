package inbox

import (
	"strconv"
	"testing"
	"time"
)

func TestContainsMention(t *testing.T) {
	tests := []struct {
		name        string
		commentText string
		username    string
		want        bool
	}{
		{
			name:        "exact @username mention",
			commentText: "Hey @alice can you review this?",
			username:    "alice",
			want:        true,
		},
		{
			name:        "@ username with space",
			commentText: "Hey @ alice can you review this?",
			username:    "alice",
			want:        true,
		},
		{
			name:        "case insensitive - uppercase in comment",
			commentText: "Hey @ALICE can you review this?",
			username:    "alice",
			want:        true,
		},
		{
			name:        "username must be lowercase - uppercase username does not match",
			commentText: "Hey @alice can you review this?",
			username:    "ALICE",
			want:        false,
		},
		{
			name:        "case insensitive - mixed case",
			commentText: "Hey @Alice can you review this?",
			username:    "alice",
			want:        true,
		},
		{
			name:        "no match - different username",
			commentText: "Hey @bob can you review this?",
			username:    "alice",
			want:        false,
		},
		{
			name:        "no match - no @ sign",
			commentText: "Hey alice can you review this?",
			username:    "alice",
			want:        false,
		},
		{
			name:        "no match - empty comment",
			commentText: "",
			username:    "alice",
			want:        false,
		},
		{
			name:        "mention at beginning of string",
			commentText: "@alice please check",
			username:    "alice",
			want:        true,
		},
		{
			name:        "mention at end of string",
			commentText: "Please check @alice",
			username:    "alice",
			want:        true,
		},
		{
			name:        "mention with space at beginning",
			commentText: "@ alice please check",
			username:    "alice",
			want:        true,
		},
		{
			name:        "username is substring of another - still matches",
			commentText: "Hey @alicewonderland what's up?",
			username:    "alice",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsMention(tt.commentText, tt.username)
			if got != tt.want {
				t.Errorf("containsMention(%q, %q) = %v, want %v",
					tt.commentText, tt.username, got, tt.want)
			}
		})
	}
}

func TestFormatMentionDate(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		// We check the output is non-empty and not the raw input for valid timestamps,
		// or equals the raw input for invalid ones.
		wantRaw bool // if true, expect the raw dateStr back
	}{
		{
			name:    "valid timestamp - returns relative time string",
			dateStr: strconv.FormatInt(time.Now().Add(-2*time.Hour).UnixMilli(), 10),
			wantRaw: false,
		},
		{
			name:    "valid timestamp - recent",
			dateStr: strconv.FormatInt(time.Now().Add(-30*time.Second).UnixMilli(), 10),
			wantRaw: false,
		},
		{
			name:    "invalid - not a number",
			dateStr: "not-a-number",
			wantRaw: true,
		},
		{
			name:    "invalid - empty string",
			dateStr: "",
			wantRaw: true,
		},
		{
			name:    "invalid - letters mixed with numbers",
			dateStr: "123abc456",
			wantRaw: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMentionDate(tt.dateStr)
			if tt.wantRaw {
				if got != tt.dateStr {
					t.Errorf("formatMentionDate(%q) = %q, want raw input %q", tt.dateStr, got, tt.dateStr)
				}
			} else {
				// For valid timestamps, the result should be a relative time string,
				// not the original numeric string.
				if got == tt.dateStr {
					t.Errorf("formatMentionDate(%q) = %q, expected a relative time string", tt.dateStr, got)
				}
				if got == "" {
					t.Errorf("formatMentionDate(%q) returned empty string", tt.dateStr)
				}
			}
		})
	}
}

func TestExtractAttachmentURLs(t *testing.T) {
	tests := []struct {
		name   string
		blocks []commentBlock
		want   int
	}{
		{
			name:   "no blocks",
			blocks: nil,
			want:   0,
		},
		{
			name: "text only blocks",
			blocks: []commentBlock{
				{Text: "hello"},
				{Text: "world"},
			},
			want: 0,
		},
		{
			name: "image attachment",
			blocks: []commentBlock{
				{Type: "image", Image: &commentMediaObject{URL: "https://example.com/image.png"}},
			},
			want: 1,
		},
		{
			name: "frame attachment",
			blocks: []commentBlock{
				{Type: "frame", Frame: &commentMediaObject{URL: "https://example.com/video.mp4"}},
			},
			want: 1,
		},
		{
			name: "mixed blocks",
			blocks: []commentBlock{
				{Text: "Check this out"},
				{Type: "image", Image: &commentMediaObject{URL: "https://example.com/a.png"}},
				{Text: "and also this"},
				{Type: "frame", Frame: &commentMediaObject{URL: "https://example.com/b.mp4"}},
				{Type: "image", Image: &commentMediaObject{URL: "https://example.com/c.png"}},
			},
			want: 3,
		},
		{
			name: "image type but nil image object",
			blocks: []commentBlock{
				{Type: "image", Image: nil},
			},
			want: 0,
		},
		{
			name: "image type but empty URL",
			blocks: []commentBlock{
				{Type: "image", Image: &commentMediaObject{URL: ""}},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAttachmentURLs(tt.blocks)
			if len(got) != tt.want {
				t.Errorf("extractAttachmentURLs() returned %d URLs, want %d", len(got), tt.want)
			}
		})
	}
}

func TestFormatMentionDate_KnownValue(t *testing.T) {
	// Use a timestamp exactly 2 hours ago. RelativeTime should return "2 hours ago".
	ts := time.Now().Add(-2 * time.Hour).UnixMilli()
	dateStr := strconv.FormatInt(ts, 10)

	got := formatMentionDate(dateStr)
	if got != "2 hours ago" {
		t.Errorf("formatMentionDate(%q) = %q, want %q", dateStr, got, "2 hours ago")
	}
}
