---
title: "clickup task edit"
description: "Auto-generated reference for clickup task edit"
---

## clickup task edit

Edit a ClickUp task

### Synopsis

Edit an existing ClickUp task.

If no task ID is provided, the command attempts to auto-detect the task ID
from the current git branch name. At least one field flag must be provided.

Custom fields can be set with --field "Name=value" (repeatable) and cleared
with --clear-field "Name" (repeatable). Use 'clickup field list' to discover
available custom fields and their types.

```
clickup task edit [<task-id>] [flags]
```

### Examples

```
  # Update status and priority
  clickup task edit --status "in progress" --priority 2

  # Edit a specific task with a custom field
  clickup task edit CU-abc123 --field "Environment=production"

  # Set due date and time estimate
  clickup task edit --due-date 2025-03-01 --time-estimate 4h

  # Clear a custom field
  clickup task edit CU-abc123 --clear-field "Environment"
```

### Options

```
      --assignee ints                 Assignee user ID(s) to add
      --clear-field stringArray       Clear a custom field value ("Name", repeatable)
      --description string            New task description
      --due-date string               Due date (YYYY-MM-DD, or "none" to clear)
      --due-date-time                 Include time component in due date
      --field stringArray             Set a custom field value ("Name=value", repeatable)
  -h, --help                          help for edit
      --links-to string               Link to another task by ID
      --markdown-description string   New task description in markdown
      --name string                   New task name
      --notify-all                    Notify all assignees and watchers
      --parent string                 Parent task ID (make this a subtask)
      --points float                  Sprint/story points (-1 to clear) (default -999)
      --priority int                  New task priority (1=Urgent, 2=High, 3=Normal, 4=Low)
      --remove-assignee ints          Assignee user ID(s) to remove
      --start-date string             Start date (YYYY-MM-DD, or "none" to clear)
      --start-date-time               Include time component in start date
      --status string                 New task status
      --tags strings                  Set tags (replaces existing)
      --time-estimate string          Time estimate (e.g. 2h, 30m, 1h30m; "0" to clear)
      --type int                      Task type (0=task, 1=milestone, or custom type ID) (default -1)
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

