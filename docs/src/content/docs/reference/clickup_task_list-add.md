---
title: "clickup task list-add"
description: "Auto-generated reference for clickup task list-add"
---

Add tasks to an additional list

### Synopsis

Add one or more tasks to an additional ClickUp list.

This does not move the task — it adds the task to the specified list
as a secondary location, so the task appears in multiple lists.

Multiple task IDs can be provided to add them all to the same list.
Each task is processed independently; errors on individual tasks are
reported but do not stop the batch.

```
clickup task list-add <task-id>... --list-id <list-id> [flags]
```

### Examples

```
  # Add a single task to a sprint list
  clickup task list-add 86abc123 --list-id 901613544162

  # Add multiple tasks to the same list
  clickup task list-add 86abc1 86abc2 86abc3 --list-id 901613544162
```

### Options

```
  -h, --help              help for list-add
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --list-id string    Target list ID (required)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

