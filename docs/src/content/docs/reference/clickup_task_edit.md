---
title: "clickup task edit"
description: "Auto-generated reference for clickup task edit"
---

Edit a ClickUp task

### Synopsis

Edit one or more existing ClickUp tasks.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. At least one field flag must be provided.

Multiple task IDs can be provided to apply the same changes to all tasks.
Each task is updated independently; errors on individual tasks are reported
but do not stop the batch.

Custom fields can be set with --field "Name=value" (repeatable) and cleared
with --clear-field "Name" (repeatable). Use 'clickup field list' to discover
available custom fields and their types.

```
clickup task edit [<task-id>...] [flags]
```

### Examples

```
  # Update status and priority (auto-detects task from git branch)
  clickup task edit --status "in progress" --priority 2

  # Edit a specific task
  clickup task edit CU-abc123 --field "Environment=production"
  clickup task edit CU-abc123 --due-date 2025-03-01 --time-estimate 4h
  clickup task edit CU-abc123 --clear-field "Environment"

  # Bulk edit: close multiple subtasks at once
  clickup task edit 86abc1 86abc2 86abc3 --status "Closed"

  # Bulk edit: set due date on many tasks
  clickup task edit 86abc1 86abc2 86abc3 --due-date 2026-03-01

  # Add tags without removing existing ones
  clickup task edit CU-abc123 --add-tags new-feature-development
  clickup task edit 86abc1 86abc2 --add-tags r&d,new-app-development

  # Remove specific tags
  clickup task edit CU-abc123 --remove-tags fix
```

### Options

```
      --add-tags strings              Add tags without removing existing ones
      --assignee ints                 Assignee user ID(s) to add
      --clear-field stringArray       Clear a custom field value ("Name", repeatable)
      --description string            New task description
      --due-date string               Due date (YYYY-MM-DD, or "none" to clear)
      --due-date-time                 Include time component in due date
      --field stringArray             Set a custom field value ("Name=value", repeatable)
  -h, --help                          help for edit
      --jq string                     Filter JSON output using a jq expression
      --json                          Output JSON
      --links-to string               Link to another task by ID
      --markdown-description string   New task description in markdown
      --name string                   New task name (convention: [Type] Context — Action (Platform))
      --notify-all                    Notify all assignees and watchers
      --parent string                 Parent task ID (make this a subtask)
      --points float                  Sprint/story points (-1 to clear) (default -999)
      --priority int                  New task priority (1=Urgent, 2=High, 3=Normal, 4=Low)
  -r, --raw                           Output raw strings instead of JSON-encoded (use with --jq)
      --remove-assignee ints          Assignee user ID(s) to remove
      --remove-tags strings           Remove specific tags
      --start-date string             Start date (YYYY-MM-DD, or "none" to clear)
      --start-date-time               Include time component in start date
      --status string                 New task status
      --tags strings                  Set tags (replaces existing)
      --template string               Format JSON output using a Go template
      --time-estimate string          Time estimate (e.g. 2h, 30m, 1h30m; "0" to clear)
      --type int                      Task type (0=task, 1=milestone, or custom type ID) (default -1)
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

