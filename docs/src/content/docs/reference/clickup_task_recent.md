---
title: "clickup task recent"
description: "Auto-generated reference for clickup task recent"
---

## clickup task recent

Show recently updated tasks

### Synopsis

Show tasks recently updated in your workspace, ordered by last activity.

By default shows your own tasks (assigned to you). Use --all to see all
recently updated tasks across the team.

Each task includes its list and folder location, making it easy to discover
where work is happening when you're unsure which list or folder to search.

```
clickup task recent [flags]
```

### Examples

```
  # Show your recent tasks
  clickup task recent

  # Show all team activity
  clickup task recent --all

  # Show more results
  clickup task recent --limit 30

  # JSON output for scripting
  clickup task recent --json
```

### Options

```
      --all               Show all team tasks, not just yours
  -h, --help              help for recent
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --limit int         Maximum number of tasks to show (default 20)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

