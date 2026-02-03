---
title: "clickup task activity"
description: "Auto-generated reference for clickup task activity"
---

## clickup task activity

View a task's details and comment history

### Synopsis

Display a task's details and all its comments in chronological order.

This gives a full picture of a task's change history by combining the task
summary (name, status, priority, assignees, dates) with every comment
posted on the task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. Branch names containing CU-<hex> or
PREFIX-<number> patterns are recognized.

```
clickup task activity [<task-id>] [flags]
```

### Examples

```
  # View activity for a specific task
  clickup task activity 86a3xrwkp

  # Auto-detect task from git branch
  clickup task activity

  # Output as JSON
  clickup task activity 86a3xrwkp --json

  # Filter with jq
  clickup task activity 86a3xrwkp --jq '.comments[] | .user'
```

### Options

```
  -h, --help              help for activity
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

