---
title: "clickup task time list"
description: "Auto-generated reference for clickup task time list"
---

## clickup task time list

View time entries for a task

### Synopsis

Display time entries logged against a ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name.

```
clickup task time list [<task-id>] [flags]
```

### Examples

```
  # List time entries for a specific task
  clickup task time list 86a3xrwkp

  # Auto-detect task from git branch
  clickup task time list

  # Output as JSON
  clickup task time list 86a3xrwkp --json

  # Filter with jq
  clickup task time list 86a3xrwkp --jq '.[] | .duration'
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup task time](/clickup-cli/reference/clickup_task_time/)	 - Track time on ClickUp tasks

