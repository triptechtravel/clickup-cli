---
title: "clickup task time list"
description: "Auto-generated reference for clickup task time list"
---

View time entries for a task or date range

### Synopsis

Display time entries logged against a ClickUp task, or query all entries
across tasks for a date range (timesheet mode).

Per-task mode (default): Shows entries for a single task. If no task ID is
provided, the CLI auto-detects it from the current git branch name.

Timesheet mode: When --start-date and --end-date are provided, shows all
time entries across tasks for the given date range. By default filters to
the current user; use --assignee to change.

```
clickup task time list [<task-id>] [flags]
```

### Examples

```
  # List time entries for a specific task
  clickup task time list 86a3xrwkp

  # Auto-detect task from git branch
  clickup task time list

  # Timesheet: all your entries for a month
  clickup task time list --start-date 2026-02-01 --end-date 2026-02-28

  # Timesheet for all workspace members
  clickup task time list --start-date 2026-02-01 --end-date 2026-02-28 --assignee all

  # Timesheet for a specific user
  clickup task time list --start-date 2026-02-01 --end-date 2026-02-28 --assignee 54695018

  # Output as JSON
  clickup task time list 86a3xrwkp --json

  # Filter with jq
  clickup task time list --start-date 2026-02-01 --end-date 2026-02-28 --jq '[.[] | {task: .task.name, hrs: (.duration | tonumber / 3600000)}]'
```

### Options

```
      --assignee string     Filter by user ID, or "all" for everyone (default: current user)
      --end-date string     End date for timesheet mode (YYYY-MM-DD)
  -h, --help                help for list
      --jq string           Filter JSON output using a jq expression
      --json                Output JSON
      --start-date string   Start date for timesheet mode (YYYY-MM-DD)
      --tag strings         Filter by task tag(s) — comma-separated or repeated (OR logic, timesheet mode only)
      --template string     Format JSON output using a Go template
```

### SEE ALSO

* [clickup task time](/clickup-cli/reference/clickup_task_time/)	 - Track time on ClickUp tasks

