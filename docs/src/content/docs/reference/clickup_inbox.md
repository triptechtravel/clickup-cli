---
title: "clickup inbox"
description: "Auto-generated reference for clickup inbox"
---

Show recent @mentions and assignments

### Synopsis

Show recent comments that @mention you and tasks newly assigned to you.

Combines two scans: a filtered call for tasks where you are an assignee
(used to detect newly assigned tasks within the lookback window) and a
workspace-wide scan for cold @mentions on tasks you are not assigned to.

Results from the workspace scan are cached locally at
~/.config/clickup/inbox_cache.json by task date_updated, so subsequent
runs only re-fetch comments for tasks that have changed. The cache
expires after 24 hours; pass --no-cache to force a full rescan (the cache
is still rewritten after a --no-cache run so subsequent runs are cheap).

Since ClickUp does not provide a public inbox API, this command
approximates your inbox by combining these two endpoints.

```
clickup inbox [flags]
```

### Examples

```
  # Show mentions and assignments from the last 7 days
  clickup inbox

  # Look back 30 days
  clickup inbox --days 30

  # Force a full rescan, ignoring the cache
  clickup inbox --no-cache

  # JSON output for scripting
  clickup inbox --json
```

### Options

```
      --days int          How many days back to search (default 7)
  -h, --help              help for inbox
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Maximum number of tasks to scan for mentions (default 200)
      --no-cache          Bypass the local cache and re-fetch comments for every task
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup](/clickup-cli/reference/clickup/)	 - ClickUp CLI - manage tasks from the command line

