---
title: "clickup task time start"
description: "Auto-generated reference for clickup task time start"
---

Start a time entry timer

### Synopsis

Start a running time entry timer in ClickUp.

Optionally associate the timer with a task. If no task ID is given, a
free-running timer is started. The task ID can be auto-detected from
the current git branch.

```
clickup task time start [<task-id>] [flags]
```

### Examples

```
  # Start a timer on a task
  clickup task time start 86abc123

  # Start with a description
  clickup task time start 86abc123 --description "Working on auth"

  # Start a free-running timer
  clickup task time start
```

### Options

```
      --billable             Mark as billable
      --description string   Timer description
  -h, --help                 help for start
      --jq string            Filter JSON output using a jq expression
      --json                 Output JSON
  -r, --raw                  Output raw strings instead of JSON-encoded (use with --jq)
      --template string      Format JSON output using a Go template
```

### SEE ALSO

* [clickup task time](/clickup-cli/reference/clickup_task_time/)	 - Track time on ClickUp tasks

