// Package api provides a generic passthrough to the ClickUp REST API.
//
// It exists so that a missing command is never a dead end. Any endpoint the
// API exposes — including ones added after this CLI was built — is reachable
// without writing Go. See docs/api-layer-plan.md for why.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/triptechtravel/clickup-cli/internal/api"
	"github.com/triptechtravel/clickup-cli/pkg/cmdutil"
)

type apiOptions struct {
	method    string
	fields    []string
	rawFields []string
	headers   []string
	input     string
	useV3     bool
	paginate  bool
	silent    bool
	jsonFlags cmdutil.JSONFlags
}

// NewCmdAPI returns the `clickup api` command.
func NewCmdAPI(f *cmdutil.Factory) *cobra.Command {
	opts := &apiOptions{}

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make an authenticated request to the ClickUp API",
		Long: `Make an authenticated HTTP request to the ClickUp API and print the response.

The endpoint is a path relative to the API root, with no leading slash —
"task/abc123", not "/api/v2/task/abc123".

The method defaults to GET, or POST when any field is supplied. Use -X to
override. Fields given with -f are type-coerced: "true"/"false" become JSON
booleans, bare numbers become numbers, "null" becomes null. Use --raw-field
to force a string.

This command covers every endpoint the API exposes, including ones this CLI
has no dedicated command for.`,
		Example: `  # Read a task
  clickup api task/abc123

  # Archive a task (no dedicated command needed)
  clickup api -X PUT task/abc123 -f archived=true

  # Un-archive it again
  clickup api -X PUT task/abc123 -f archived=false

  # List tasks in a list, extracting names
  clickup api list/901234/task --jq '.tasks[].name'

  # Force a string value rather than a boolean
  clickup api -X PUT task/abc123 --raw-field name=true

  # Send a body from a file, or stdin
  clickup api -X POST list/901234/task --input body.json
  echo '{"name":"New task"}' | clickup api -X POST list/901234/task --input -

  # Hit the v3 API
  clickup api --v3 workspaces/123/docs`,
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.NeedsAuth(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAPI(f, opts, args[0])
		},
	}

	cmd.Flags().StringVarP(&opts.method, "method", "X", "", "HTTP method (default GET, or POST when fields are given)")
	cmd.Flags().StringArrayVarP(&opts.fields, "field", "f", nil, "Body field as key=value, type-coerced (repeatable)")
	cmd.Flags().StringArrayVar(&opts.rawFields, "raw-field", nil, "Body field as key=value, always a string (repeatable)")
	cmd.Flags().StringArrayVarP(&opts.headers, "header", "H", nil, "Extra header as key:value (repeatable)")
	cmd.Flags().StringVar(&opts.input, "input", "", `Read the request body from a file, or "-" for stdin`)
	cmd.Flags().BoolVar(&opts.useV3, "v3", false, "Use the v3 API base URL")
	cmd.Flags().BoolVar(&opts.paginate, "paginate", false,
		"Follow pagination and merge every page (collection endpoints only)")
	cmd.Flags().BoolVar(&opts.silent, "silent", false, "Do not print the response body")
	cmdutil.AddJSONFlags(cmd, &opts.jsonFlags)

	return cmd
}

func runAPI(f *cmdutil.Factory, opts *apiOptions, endpoint string) error {
	client, err := f.ApiClient()
	if err != nil {
		return err
	}

	body, err := buildBody(opts)
	if err != nil {
		return err
	}

	method := strings.ToUpper(opts.method)
	if method == "" {
		if body != nil {
			method = http.MethodPost
		} else {
			method = http.MethodGet
		}
	}

	base := client.BaseURL()
	if opts.useV3 {
		base = client.BaseURLV3()
	}
	url := base + "/" + strings.TrimLeft(endpoint, "/")

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, h := range opts.headers {
		k, v, ok := strings.Cut(h, ":")
		if !ok {
			return fmt.Errorf("invalid --header %q: want key:value", h)
		}
		req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
	}

	resp, err := client.DoRequest(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Non-2xx must not exit 0. Surface the API's own error text, which carries
	// the ECODE that makes ClickUp errors diagnosable.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}

	if opts.silent {
		return nil
	}

	// ClickUp paginates several collection endpoints and signals more pages with
	// "last_page": false. Returning page 0 without saying so would present a
	// partial answer as a complete one — the same failure this CLI has already
	// shipped once.
	if hasMorePages(respBody) {
		if !opts.paginate {
			_, _ = fmt.Fprintf(f.IOStreams.ErrOut,
				"! more pages available; this is page %d only. Re-run with --paginate for all of them.\n",
				pageOf(url))
		} else {
			respBody, err = followPages(client, req, respBody, url)
			if err != nil {
				return err
			}
		}
	}

	return writeResponse(f, opts, respBody)
}

// followPages walks subsequent pages and merges their array fields into the
// first page's document, so the caller sees one combined result.
// maxPages bounds the walk. A server that always reports last_page:false —
// through a bug, a proxy, or an endpoint that does not really paginate — would
// otherwise loop forever against a live API.
const maxPages = 500

func followPages(client *api.Client, first *http.Request, firstBody []byte, url string) ([]byte, error) {
	// Subsequent pages are re-issued without a body, so a request that carried
	// one cannot be paginated correctly. Refuse rather than silently send an
	// empty body and merge whatever comes back.
	if first.Method != http.MethodGet {
		return nil, fmt.Errorf("--paginate only supports GET; %s requests may carry a body that cannot be replayed", first.Method)
	}

	merged := map[string]any{}
	if err := json.Unmarshal(firstBody, &merged); err != nil {
		// Not an object — nothing sensible to merge into.
		return firstBody, nil
	}

	start := pageOf(url) + 1
	for page := start; ; page++ {
		if page-start >= maxPages {
			return nil, fmt.Errorf("stopped after %d pages: the API never reported last_page", maxPages)
		}
		next, err := withPage(url, page)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(first.Context(), first.Method, next, nil)
		if err != nil {
			return nil, err
		}
		req.Header = first.Header.Clone()

		resp, err := client.DoRequest(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("HTTP %d while paginating page %d: %s",
				resp.StatusCode, page, strings.TrimSpace(string(body)))
		}

		var doc map[string]any
		if err := json.Unmarshal(body, &doc); err != nil {
			return nil, fmt.Errorf("page %d is not a JSON object: %w", page, err)
		}
		for k, v := range doc {
			arr, ok := v.([]any)
			if !ok {
				continue
			}
			if existing, ok := merged[k].([]any); ok {
				merged[k] = append(existing, arr...)
			}
		}
		merged["last_page"] = true

		if !hasMorePages(body) {
			break
		}
	}

	return json.Marshal(merged)
}

// withPage returns url with its page query parameter set to n.
func withPage(rawURL string, n int) (string, error) {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("page", strconv.Itoa(n))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// hasMorePages reports whether the response explicitly says it is not the last
// page. Absent or true means there is nothing to warn about.
func hasMorePages(body []byte) bool {
	var probe struct {
		LastPage *bool `json:"last_page"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return false
	}
	return probe.LastPage != nil && !*probe.LastPage
}

// pageOf extracts the page query parameter, defaulting to 0.
func pageOf(rawURL string) int {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(u.Query().Get("page"))
	if err != nil {
		return 0
	}
	return n
}

// writeResponse applies --jq/--template when asked, and otherwise prints the
// body through unchanged so non-JSON responses survive.
func writeResponse(f *cmdutil.Factory, opts *apiOptions, respBody []byte) error {
	out := f.IOStreams.Out

	if opts.jsonFlags.WantsJSON() {
		var parsed any
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return fmt.Errorf("response is not JSON, cannot apply --jq/--template: %w", err)
		}
		return opts.jsonFlags.OutputJSON(out, parsed)
	}

	if _, err := out.Write(respBody); err != nil {
		return err
	}
	if len(respBody) > 0 && !bytes.HasSuffix(respBody, []byte("\n")) {
		_, err := fmt.Fprintln(out)
		return err
	}
	return nil
}

// buildBody assembles the JSON request body from --field/--raw-field/--input.
func buildBody(opts *apiOptions) ([]byte, error) {
	if opts.input != "" {
		if len(opts.fields) > 0 || len(opts.rawFields) > 0 {
			return nil, fmt.Errorf("--input cannot be combined with --field or --raw-field")
		}
		if opts.input == "-" {
			return io.ReadAll(os.Stdin)
		}
		return os.ReadFile(opts.input)
	}

	if len(opts.fields) == 0 && len(opts.rawFields) == 0 {
		return nil, nil
	}

	payload := map[string]any{}
	for _, raw := range opts.rawFields {
		k, v, ok := strings.Cut(raw, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --raw-field %q: want key=value", raw)
		}
		payload[k] = v
	}
	for _, raw := range opts.fields {
		k, v, ok := strings.Cut(raw, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --field %q: want key=value", raw)
		}
		payload[k] = coerce(v)
	}
	return json.Marshal(payload)
}

// coerce turns a flag string into the JSON type it obviously is. This is what
// makes `-f archived=true` send a boolean rather than the string "true", which
// ClickUp silently ignores.
func coerce(v string) any {
	switch v {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}
	if fl, err := strconv.ParseFloat(v, 64); err == nil {
		return fl
	}
	// JSON literals (objects, arrays) pass through structured.
	if strings.HasPrefix(v, "{") || strings.HasPrefix(v, "[") {
		var parsed any
		if err := json.Unmarshal([]byte(v), &parsed); err == nil {
			return parsed
		}
	}
	return v
}
