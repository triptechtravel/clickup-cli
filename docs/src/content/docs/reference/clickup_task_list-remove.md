---
title: "clickup task list-remove"
description: "Auto-generated reference for clickup task list-remove"
---

Remove tasks from a list

### Synopsis

Remove one or more tasks from a ClickUp list.

This removes the task's membership in the specified list. The task must
belong to at least one other list — you cannot remove a task from its
only list.

Multiple task IDs can be provided to remove them all from the same list.
Each task is processed independently; errors on individual tasks are
reported but do not stop the batch.

```
clickup task list-remove <task-id>... --list-id <list-id> [flags]
```

### Examples

```
  # Remove a single task from a sprint list
  clickup task list-remove 86abc123 --list-id 901613544162

  # Remove multiple tasks from the same list
  clickup task list-remove 86abc1 86abc2 86abc3 --list-id 901613544162
```

### Options

```
  -h, --help              help for list-remove
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --list-id string    List ID to remove from (required)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

