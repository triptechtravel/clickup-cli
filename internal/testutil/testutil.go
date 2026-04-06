// Package testutil provides shared test infrastructure for clickup-cli.
//
// TestFactory wires an httptest.Server with a Factory that has all
// dependencies overridden, so command tests can run without real auth,
// config, or API access.
package testutil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/internal/config"
	"github.com/triptechtravel/clickup-cli/internal/iostreams"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

// TestFactory bundles everything needed for command-level tests.
type TestFactory struct {
	Factory *cmdutil.Factory
	IOS     *iostreams.IOStreams
	OutBuf  *bytes.Buffer
	ErrBuf  *bytes.Buffer
	Server  *httptest.Server
	Mux     *http.ServeMux
}

// NewTestFactory creates a fully wired test environment:
//   - httptest.Server with a mux for registering mock endpoints
//   - api.Client pointing at that server
//   - Factory with API client + config overrides
//   - Captured stdout/stderr buffers for assertions
func NewTestFactory(t *testing.T) *TestFactory {
	t.Helper()

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(strings.NewReader("")),
		Out:    outBuf,
		ErrOut: errBuf,
	}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := api.NewTestClient(server.URL)

	f := cmdutil.NewFactory(ios)
	f.SetAPIClient(client)
	f.SetConfig(&config.Config{
		Workspace:    "12345",
		Space:        "67890",
		SprintFolder: "",
	})

	return &TestFactory{
		Factory: f,
		IOS:     ios,
		OutBuf:  outBuf,
		ErrBuf:  errBuf,
		Server:  server,
		Mux:     mux,
	}
}

// Handle registers a handler for the given method + API v2 path.
// The path is relative to /api/v2/ (e.g., "task/abc123").
func (tf *TestFactory) Handle(method, path string, status int, body string) {
	fullPath := "/api/v2/" + strings.TrimLeft(path, "/")
	tf.Mux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
}

// HandleV3 registers a handler for the given method + API v3 path.
// The path is relative to /api/v3/ (e.g., "workspaces/123/docs").
func (tf *TestFactory) HandleV3(method, path string, status int, body string) {
	fullPath := "/api/v3/" + strings.TrimLeft(path, "/")
	tf.Mux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "99")
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
}

// HandleFunc registers a custom handler for the given API v2 path.
func (tf *TestFactory) HandleFunc(path string, handler http.HandlerFunc) {
	fullPath := "/api/v2/" + strings.TrimLeft(path, "/")
	tf.Mux.HandleFunc(fullPath, handler)
}

// HandleFuncV3 registers a custom handler for the given API v3 path.
func (tf *TestFactory) HandleFuncV3(path string, handler http.HandlerFunc) {
	fullPath := "/api/v3/" + strings.TrimLeft(path, "/")
	tf.Mux.HandleFunc(fullPath, handler)
}

// RunCommand executes a cobra command with the given args and returns the error.
func RunCommand(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.SetArgs(args)
	return cmd.Execute()
}
