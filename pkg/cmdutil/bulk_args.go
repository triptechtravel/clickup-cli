package cmdutil

import "strings"

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
