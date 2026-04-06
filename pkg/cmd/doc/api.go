package doc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// validParentTypes are the accepted string values for --parent-type.
var validParentTypes = map[string]int{
	"SPACE":      4,
	"FOLDER":     5,
	"LIST":       6,
	"WORKSPACE":  7,
	"EVERYTHING": 12,
}

// validVisibility are the accepted string values for --visibility.
var validVisibility = []string{"PUBLIC", "PRIVATE", "PERSONAL", "HIDDEN"}

// validContentFormats are the accepted values for --content-format.
var validContentFormats = []string{"text/md", "text/plain"}

// validEditModes are the accepted values for --content-edit-mode.
var validEditModes = []string{"replace", "append", "prepend"}

// resolveWorkspaceID returns the workspace ID from config.
func resolveWorkspaceID(f *cmdutil.Factory) (string, error) {
	cfg, err := f.Config()
	if err != nil {
		return "", err
	}
	if cfg.Workspace == "" {
		return "", fmt.Errorf("no workspace configured; run 'clickup auth login' to set it up")
	}
	return cfg.Workspace, nil
}

// parseParentType converts a string like "SPACE" or "4" to the int type used by the API.
func parseParentType(s string) (int, error) {
	upper := strings.ToUpper(s)
	if v, ok := validParentTypes[upper]; ok {
		return v, nil
	}
	// Try numeric fallback
	var n int
	if _, err := fmt.Sscan(s, &n); err == nil {
		return n, nil
	}
	keys := make([]string, 0, len(validParentTypes))
	for k := range validParentTypes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return 0, fmt.Errorf("invalid parent type %q; valid values: %s", s, strings.Join(keys, "|"))
}

// containsString returns true if s is in the slice (case-insensitive).
func containsString(slice []string, s string) bool {
	upper := strings.ToUpper(s)
	for _, v := range slice {
		if strings.ToUpper(v) == upper {
			return true
		}
	}
	return false
}
