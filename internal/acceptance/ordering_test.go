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
	//
	// Both spellings, because the first version of this guard matched only the
	// query-string literal and so was blind to pkg/cmdutil/recent.go, which
	// says the same thing with struct fields — the one call site that had been
	// returning four-year-old tasks from a command called "recent". A guard
	// written to catch "the call site nobody thought about" has to match the
	// behaviour, not one way of spelling it.
	var offenders []string
	offenders = append(offenders, grepRepo(t, ".", `order_by=updated.*reverse=true`)...)
	offenders = append(offenders, grepRepo(t, ".", `reverse=true.*order_by=updated`)...)
	offenders = append(offenders, grepRepo(t, ".", `(?i)Reverse:\s*true`)...)

	assert.Empty(t, offenders,
		"these read the workspace oldest-first; ClickUp treats reverse=true as ascending, and every one of these reads a prefix")
}

// The commands that resolve or list tasks by recency must ask for newest
// first. Asserted through the real command, so a regression expressed any way
// at all is caught.
func TestOrdering_RecentTasksAreActuallyRecent(t *testing.T) {
	var (
		mu   sync.Mutex
		urls []string
	)
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100,"username":"isaac"}}`)
	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		urls = append(urls, r.URL.String())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"tasks":[]}`))
	})

	_, _, _ = runCLI(t, tf, "task", "recent")

	require.NotEmpty(t, urls, "no team task requests recorded")
	for _, u := range urls {
		assert.Contains(t, u, "order_by=updated")
		assert.NotContains(t, u, "reverse=true",
			"`task recent` asked for oldest-first: %s", u)
	}
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
