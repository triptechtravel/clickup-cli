---
title: "clickup task time log"
description: "Auto-generated reference for clickup task time log"
---

## clickup task time log

Log time to a task

### Synopsis

Log a time entry against a ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name.

```
clickup task time log [<task-id>] [flags]
```

### Examples

```
  # Log 2 hours to a specific task
  clickup task time log 86a3xrwkp --duration 2h

  # Log 1h30m with a description
  clickup task time log --duration 1h30m --description "Implemented auth flow"

  # Log time for a specific date
  clickup task time log 86a3xrwkp --duration 45m --date 2025-01-15

  # Log billable time
  clickup task time log --duration 3h --billable
```

### Options

```
      --billable             Mark time entry as billable
      --date string          Date of the work (YYYY-MM-DD, default today)
      --description string   Description of work done
      --duration string      Duration to log (e.g. "2h", "30m", "1h30m")
  -h, --help                 help for log
```

### SEE ALSO

* [clickup task time](/clickup-cli/reference/clickup_task_time/)	 - Track time on ClickUp tasks

