---
title: "clickup api"
description: "Auto-generated reference for clickup api"
---

Make an authenticated request to the ClickUp API

### Synopsis

Make an authenticated HTTP request to the ClickUp API and print the response.

The endpoint is a path relative to the API root, with no leading slash —
"task/abc123", not "/api/v2/task/abc123".

The method defaults to GET, or POST when any field is supplied. Use -X to
override. Fields given with -f are type-coerced: "true"/"false" become JSON
booleans, bare numbers become numbers, "null" becomes null. Use --raw-field
to force a string.

This command covers every endpoint the API exposes, including ones this CLI
has no dedicated command for.

```
clickup api <endpoint> [flags]
```

### Examples

```
  # Read a task
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
  clickup api --v3 workspaces/123/docs
```

### Options

```
  -f, --field stringArray       Body field as key=value, type-coerced (repeatable)
  -H, --header stringArray      Extra header as key:value (repeatable)
  -h, --help                    help for api
      --input string            Read the request body from a file, or "-" for stdin
      --jq string               Filter JSON output using a jq expression
      --json                    Output JSON
  -X, --method string           HTTP method (default GET, or POST when fields are given)
      --paginate                Follow pagination and merge every page (collection endpoints only)
  -r, --raw                     Output raw strings instead of JSON-encoded (use with --jq)
      --raw-field stringArray   Body field as key=value, always a string (repeatable)
      --silent                  Do not print the response body
      --template string         Format JSON output using a Go template
      --v3                      Use the v3 API base URL
```

### SEE ALSO

* [clickup](/clickup-cli/reference/clickup/)	 - ClickUp CLI - manage tasks from the command line

