package task

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/triptechtravel/clickup-cli/internal/testutil"
)

// assigneeMux serves the team endpoint, splitting assignee-filtered requests
// from unfiltered ones so a dropped filter is visible.
func assigneeMux(tf *testutil.TestFactory, filtered map[int]string, unfiltered map[int]string) *[]string {
	var mu sync.Mutex
	var urls []string
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)
	tf.Handle("GET", "team", 200, `{"teams":[{"id":"12345","name":"W","members":[{"user":{"id":100,"username":"alice"}},{"user":{"id":200,"username":"bob"}}]}]}`)
	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		urls = append(urls, r.URL.String())
		mu.Unlock()
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		table := unfiltered
		if strings.Contains(r.URL.RawQuery, "assignees") {
			table = filtered
		}
		body := `{"tasks":[]}`
		if b, ok := table[page]; ok {
			body = b
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(body))
	})
	return &urls
}

// The assignee-only branch read exactly one page and said nothing about it, so
// someone with 250 open tasks was shown 100 and told nothing.
func TestSearch_AssigneeOnlyPaginatesAndDiscloses(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	assigneeMux(tf, map[int]string{
		0: searchTasksJSON(fillerPairs(100)...),
		1: searchTasksJSON([2]string{"deep", "Task on page two"}),
	}, nil)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "--assignee", "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.OutBuf.String(), "Task on page two", "assignee listing stopped at page 0")
}

// The space walk never reads opts.assignee, so falling back to it printed other
// people's tasks as the requested assignee's — a wrong answer, not a missing
// one, with stderr confirming the filter that had just been discarded.
func TestSearch_AssigneeFilterSurvivesTheFallback(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	assigneeMux(tf, nil, nil) // assignee sweep finds nothing

	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[{"id":"sp1","name":"S"}]}`)
	tf.Handle("GET", "space/sp1/folder", 200, `{"folders":[]}`)
	tf.Handle("GET", "space/sp1/list", 200, `{"lists":[{"id":"L1","name":"L"}]}`)
	tf.HandleFunc("list/L1/task", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		if r.URL.Query().Get("page") == "0" {
			_, _ = w.Write([]byte(`{"tasks":[{"id":"BOB1","name":"payments bug","status":{"status":"open"},"assignees":[{"username":"bob"}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"tasks":[]}`))
	})

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "payments", "--assignee", "alice", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.NotContains(t, tf.OutBuf.String(), "BOB1",
		"bob's task reported as alice's; the fallback dropped the assignee filter")
}

// Discovery failures dropped whole spaces with no counter and no disclosure —
// a 500 on one folder listing silently removed every list in that space from
// the path advertised as reaching everything.
func TestSpaceWalk_DisclosesDiscoveryFailures(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[{"id":"sp1","name":"Space One"}]}`)
	tf.Handle("GET", "space/sp1/folder", 500, `{"err":"boom"}`)
	tf.Handle("GET", "space/sp1/list", 500, `{"err":"boom"}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--space", "Space One", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.ErrBuf.String(), "Not shown:",
		"a space that could not be opened was passed over in silence")
}

// A filter that matches no container is not the same as a query that matches no
// task, and reporting the second when the first happened sends the user looking
// for a task that is sitting right there.
func TestSpaceWalk_SaysWhenTheFilterMatchedNothing(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[{"id":"sp1","name":"Development"}]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--space", "Devlopment"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.ErrBuf.String(), "no space matched",
		"a typo'd --space read as 'this task does not exist'")
}

// --exact is documented as suppressing fuzzy results; it must not report the
// suppression as absence.
func TestSearch_ExactDisclosesWhatItSuppressed(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)
	sweepMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"a", "Deployment Pipeline"}),
	}, `{"tasks":[]}`)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "dplyment", "--exact", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Contains(t, tf.ErrBuf.String(), "Not shown:", "--exact dropped matches silently")
}

// The --exact filter ran before the per-word retry appended to the results, so
// a flag documented as "no fuzzy results" printed rows whose own MATCH column
// said fuzzy.
func TestSearch_ExactAlsoAppliesToTheWordRetry(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)
	sweepMux(tf, map[int]string{
		0: searchTasksJSON([2]string{"a", "Deployment Pipeline"}),
	}, `{"tasks":[]}`)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "dplyment qqqqzz", "--exact", "--no-cache"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.NotContains(t, tf.OutBuf.String(), "fuzzy",
		"--exact printed a fuzzy row that the word retry added after the filter ran")
}

// Matches already in hand must survive a later page failing. The index path
// keeps what it fetched; the sweep threw it all away and exited non-zero.
func TestSearch_SweepKeepsMatchesFoundBeforeAnError(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)
	tf.HandleFunc("team/12345/task", func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page >= 2 {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"err":"server exploded"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(searchTasksJSON(append(fillerPairs(99), [2]string{
			"m" + strconv.Itoa(page), "5.6.1 match on page " + strconv.Itoa(page),
		})...)))
	})
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--no-cache", "--exact"); err != nil {
		t.Fatalf("a mid-sweep error should degrade, not fail: %v", err)
	}

	out := tf.OutBuf.String()
	assert.Contains(t, out, "5.6.1 match on page 0")
	assert.Contains(t, out, "5.6.1 match on page 1")
	assert.Contains(t, tf.ErrBuf.String(), "Not shown:", "lost pages not disclosed")
}

// Exactly filling the page budget is not evidence of more behind it. A
// workspace holding exactly maxSweepPages*100 tasks was told, on every single
// search, that results had been withheld.
func TestSearch_NoFalseTruncationWhenTheCorpusEndsOnACapBoundary(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)
	pages := map[int]string{}
	for i := 0; i < maxSweepPages; i++ {
		pages[i] = searchTasksJSON(append(fillerPairs(99), [2]string{
			"m" + strconv.Itoa(i), "5.6.1 card " + strconv.Itoa(i),
		})...)
	}
	sweepMux(tf, pages, `{"tasks":[]}`) // page maxSweepPages returns empty

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "5.6.1", "--no-cache", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.NotContains(t, tf.ErrBuf.String(), "Not shown:",
		"claimed truncation on a corpus that ended exactly on the cap")
}

// The remedy has to be one the user can act on. --comments and --assignee
// bypass the index too, so telling those callers to drop --no-cache — which
// they never passed — is advice that changes nothing.
func TestSearch_CapMessageDoesNotSuggestDroppingAFlagNotPassed(t *testing.T) {
	tf := testutil.NewTestFactory(t)
	tf.Handle("GET", "user", 200, `{"user":{"id":100}}`)
	full := searchTasksJSON(fillerPairs(100)...)
	pages := map[int]string{}
	for i := 0; i < maxSweepPages+1; i++ {
		pages[i] = full
	}
	sweepMux(tf, pages, `{"tasks":[]}`)
	tf.Handle("GET", "team/12345/space", 200, `{"spaces":[]}`)
	tf.Handle("GET", "task/filler0/comment", 200, `{"comments":[]}`)

	cmd := NewCmdSearch(tf.Factory)
	if err := testutil.RunCommand(t, cmd, "zzznomatch", "--comments", "--exact"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.NotContains(t, tf.ErrBuf.String(), "Drop --no-cache",
		"told the user to drop a flag they never passed")
}
