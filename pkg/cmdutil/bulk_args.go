package cmdutil

import (
	"fmt"
	"strings"
)

// ExpandIDArgs normalizes positional bulk-ID arguments. Shells differ on
// whether unquoted variables word-split (bash splits on IFS by default; zsh
// does not), so a user running `clickup task delete $IDS` may end up passing
// "id1 id2 id3" as a single argument with embedded whitespace. The CLI then
// URL-encodes the spaces and the API rejects the request.
//
// Splitting each arg on any ASCII whitespace handles that case, plus the
// common pattern of feeding a file/stdout containing newline- or
// space-separated IDs. Empty fields are dropped.
func ExpandIDArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		for _, part := range strings.Fields(a) {
			out = append(out, part)
		}
	}
	return out
}

// ValidateTaskIDArgs rejects IDs that contain characters which would be
// silently URL-encoded into a path segment (whitespace, slashes, query
// markers). A clear "invalid task ID" error is more useful than the
// downstream API's generic 401/404 reaction to a mangled URL.
func ValidateTaskIDArgs(ids []string) error {
	for _, id := range ids {
		if id == "" {
			return fmt.Errorf("empty task ID")
		}
		if strings.ContainsAny(id, " \t\n\r/?#&") {
			return fmt.Errorf("invalid task ID %q: contains whitespace or URL-special characters", id)
		}
	}
	return nil
}
