---
title: "clickup inbox"
description: "Auto-generated reference for clickup inbox"
---

## clickup inbox

Show recent @mentions

### Synopsis

Show recent comments that @mention you across your workspace.

Scans recently updated tasks for comments containing your username.
Since ClickUp does not provide a public inbox API, this command
approximates your inbox by searching task comments.

```
clickup inbox [flags]
```

### Examples

```
  # Show mentions from the last 7 days
  clickup inbox

  # Look back 30 days
  clickup inbox --days 30

  # JSON output for scripting
  clickup inbox --json
```

### Options

```
      --days int          How many days back to search (default 7)
  -h, --help              help for inbox
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Maximum number of tasks to scan for mentions (default 50)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup](/clickup-cli/reference/clickup/)	 - ClickUp CLI - manage tasks from the command line

