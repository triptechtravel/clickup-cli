---
title: "clickup task view"
description: "Auto-generated reference for clickup task view"
---

## clickup task view

View a ClickUp task

### Synopsis

Display detailed information about a ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. Branch names containing CU-<id> or
PREFIX-<number> patterns are recognized.

```
clickup task view [<task-id>] [flags]
```

### Examples

```
  # View a specific task
  clickup task view 86a3xrwkp

  # Auto-detect task from git branch
  clickup task view

  # Output as JSON
  clickup task view 86a3xrwkp --json
```

### Options

```
  -h, --help              help for view
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

