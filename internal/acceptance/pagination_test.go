package acceptance

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

// A paginated endpoint must not be presented as complete — the same class of
// bug as the multi-list blindness, in a different place.
func TestIncident_PassthroughDoesNotSilentlyTruncate(t *testing.T) {
	t.Run("warns when more pages exist", func(t *testing.T) {
		tf := testutil.NewTestFactory(t)
		tf.HandleFunc("list/1/task", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") == "" {
				fmt.Fprint(w, `{"tasks":[{"id":"a"}],"last_page":false}`)
				return
			}
			fmt.Fprint(w, `{"tasks":[{"id":"b"}],"last_page":true}`)
		})
		_, errOut, err := runCLI(t, tf, "api", "list/1/task")
		require.NoError(t, err)
		assert.Regexp(t, `(?i)more pages`, errOut, "must disclose truncation")
		assert.Contains(t, errOut, "--paginate")
	})

	t.Run("--paginate merges every page", func(t *testing.T) {
		tf := testutil.NewTestFactory(t)
		tf.HandleFunc("list/1/task", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") == "" {
				fmt.Fprint(w, `{"tasks":[{"id":"a"}],"last_page":false}`)
				return
			}
			fmt.Fprint(w, `{"tasks":[{"id":"b"}],"last_page":true}`)
		})
		out, _, err := runCLI(t, tf, "api", "list/1/task", "--paginate", "--jq", ".tasks[].id")
		require.NoError(t, err)
		assert.Contains(t, out, "a")
		assert.Contains(t, out, "b", "page 2 must be merged in, not dropped")
	})
}
