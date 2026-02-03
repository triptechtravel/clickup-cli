---
title: "clickup task create"
description: "Auto-generated reference for clickup task create"
---

## clickup task create

Create a new ClickUp task

### Synopsis

Create a new task in a ClickUp list.

The --list-id flag is required to specify which list to create the task in.
If --name is not provided, the command enters interactive mode and prompts
for the task name, description, status, priority, due date, and time estimate.

Additional properties can be set with flags:
  --tags           Tags to add (comma-separated or repeat flag)
  --due-date       Due date in YYYY-MM-DD format
  --start-date     Start date in YYYY-MM-DD format
  --time-estimate  Time estimate (e.g. "2h", "30m", "1h30m")
  --points         Sprint/story points
  --field          Set a custom field ("Name=value", repeatable)
  --parent         Create as subtask of another task
  --type           Task type (0=task, 1=milestone)

```
clickup task create [flags]
```

### Examples

```
  # Create with flags
  clickup task create --list-id 12345 --name "Fix login bug" --priority 2

  # Interactive mode (prompts for details)
  clickup task create --list-id 12345

  # Create with custom field and due date
  clickup task create --list-id 12345 --name "Deploy v2" --field "Environment=staging" --due-date 2025-03-01

  # Create a subtask
  clickup task create --list-id 12345 --name "Write tests" --parent 86abc123
```

### Options

```
      --assignee ints                 Assignee user ID(s)
      --description string            Task description
      --due-date string               Due date (YYYY-MM-DD)
      --due-date-time                 Include time component in due date
      --field stringArray             Set a custom field value ("Name=value", repeatable)
  -h, --help                          help for create
      --links-to string               Link to another task by ID
      --list-id string                ClickUp list ID (required)
      --markdown-description string   Task description in markdown
      --name string                   Task name
      --notify-all                    Notify all assignees and watchers
      --parent string                 Parent task ID (create as subtask)
      --points float                  Sprint/story points (default -999)
      --priority int                  Task priority (1=Urgent, 2=High, 3=Normal, 4=Low)
      --start-date string             Start date (YYYY-MM-DD)
      --start-date-time               Include time component in start date
      --status string                 Task status
      --tags strings                  Tags to add to the task
      --time-estimate string          Time estimate (e.g. 2h, 30m, 1h30m)
      --type int                      Task type (0=task, 1=milestone, or custom type ID) (default -1)
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

