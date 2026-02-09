package link

import (
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// upsertLink stores the link entry in the task description.
func upsertLink(f *cmdutil.Factory, taskID string, entry linkEntry) error {
	return upsertDescriptionLinks(f, taskID, entry)
}
