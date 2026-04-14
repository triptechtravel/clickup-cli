---
title: "clickup task view"
description: "Auto-generated reference for clickup task view"
---

View one or more ClickUp tasks

### Synopsis

Display detailed information about one or more ClickUp tasks.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. Branch names containing CU-<id> or
PREFIX-<number> patterns are recognized.

If no task ID is found in the branch name, the command checks for an
associated GitHub PR and searches task descriptions for the PR URL.

Multiple task IDs can be provided for bulk fetching. In bulk mode, tasks are
fetched concurrently (up to 10 parallel requests). Bulk mode requires JSON
output (--json, --jq, or --template) and returns an array of tasks.

```
clickup task view [<task-id>...] [flags]
```

### Examples

```
  # View a specific task
  clickup task view 86a3xrwkp

  # Auto-detect task from git branch
  clickup task view

  # Output as JSON (includes subtasks with IDs, dates, and statuses)
  clickup task view 86a3xrwkp --json

  # View with recursive subtasks (fetches all descendants)
  clickup task view 86a3xrwkp --recursive --json

  # Bulk fetch multiple tasks as JSON array
  clickup task view 86abc1 86abc2 86abc3 --json

  # Extract tags from multiple tasks
  clickup task view 86abc1 86abc2 --jq '.[].tags[].name'

  # Extract subtask IDs for bulk operations
  clickup task view 86parent --json  # then use .subtasks[].id
```

### Options

```
  -h, --help              help for view
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --recursive         Recursively fetch all descendant subtasks
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

