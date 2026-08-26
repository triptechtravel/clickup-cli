package acceptance

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

// ClickUp reads `reverse=true` on `order_by=updated` as *ascending*. Every
// caller of the team task endpoint reads only a prefix of the result — five
// pages here, ten there, a page cap in the index sync — so asking for ascending
// order points the read at the oldest tasks in the workspace and makes recent
// ones, the ones anyone is actually looking for, the last thing found.
//
// This was fixed once in task search and left in place in two siblings. A
// structural guard is the only thing that catches the next one: a unit test
// per call site would not have been written for the call site nobody thought
// about.
func TestOrdering_NoPrefixReaderAsksForOldestFirst(t *testing.T) {
	// grepRepo yields "path:line", not line content, so the whole condition has
	// to live in the pattern.
	offenders := grepRepo(t, ".", `order_by=updated&reverse=true`)

	assert.Empty(t, offenders,
		"these read the workspace oldest-first; drop reverse=true so recent tasks are reachable")
}

// The same guard, from the other side: exercise the two commands that resolve a
// task by name and assert they ask for newest-first.
func TestOrdering_TaskLookupsRequestNewestFirst(t *testing.T) {
	var (
		mu   sync.Mutex
		urls []string
	)
	record := func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		urls = append(urls, r.URL.String())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"tasks":[]}`))
	}

	tf := testutil.NewTestFactory(t)
	tf.HandleFunc("team/12345/task", record)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[]}`)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)

	_, _, _ = runCLI(t, tf, "task", "search", "some task name", "--no-cache")
	require.NotEmpty(t, urls, "no team task requests recorded")

	for _, u := range urls {
		assert.Contains(t, u, "order_by=updated")
		assert.NotContains(t, u, "reverse=true")
	}
}
