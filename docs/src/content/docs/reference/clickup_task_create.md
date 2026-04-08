---
title: "clickup task create"
description: "Auto-generated reference for clickup task create"
---

Create a new ClickUp task

### Synopsis

Create a new task in a ClickUp list.

Either --list-id or --current is required. Use --current to automatically
resolve the active sprint list from the configured sprint folder.
If --name is not provided, the command enters interactive mode and prompts
for the task name, description, status, priority, due date, and time estimate.

Use --from-file to bulk create tasks from a JSON file. The file should
contain an array of task objects. Each object supports the same fields as
the CLI flags.

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
  # Create in the current sprint (auto-resolves list from sprint folder)
  clickup task create --current \
    --name "[Bug] Auth — Fix login timeout (API)" --priority 2

  # Create with explicit list ID
  clickup task create --list-id 12345 \
    --name "[Bug] Auth — Fix login timeout (API)" --priority 2

  # Interactive mode (prompts for details)
  clickup task create --current

  # Create with custom field and due date
  clickup task create --current \
    --name "[Feature] Deploy — Release v2 to staging" \
    --field "Environment=staging" --due-date 2025-03-01

  # Create a subtask
  clickup task create --list-id 12345 --name "Write tests" --parent 86abc123

  # Bulk create from JSON file
  clickup task create --current --from-file tasks.json

  # Bulk create subtasks under a parent
  clickup task create --list-id 12345 --from-file checklist.json
```

### Options

```
      --assignee ints                 Assignee user ID(s)
      --current                       Create in the current sprint (auto-resolves list ID from sprint folder)
      --description string            Task description
      --due-date string               Due date (YYYY-MM-DD)
      --due-date-time                 Include time component in due date
      --field stringArray             Set a custom field value ("Name=value", repeatable)
      --from-file string              Create tasks from a JSON file (array of task objects)
  -h, --help                          help for create
      --jq string                     Filter JSON output using a jq expression
      --json                          Output JSON
      --links-to string               Link to another task by ID
      --list-id string                ClickUp list ID
      --markdown-description string   Task description in markdown
      --name string                   Task name (convention: [Type] Context — Action (Platform))
      --notify-all                    Notify all assignees and watchers
      --parent string                 Parent task ID (create as subtask)
      --points float                  Sprint/story points (default -999)
      --priority int                  Task priority (1=Urgent, 2=High, 3=Normal, 4=Low)
  -r, --raw                           Output raw strings instead of JSON-encoded (use with --jq)
      --start-date string             Start date (YYYY-MM-DD)
      --start-date-time               Include time component in start date
      --status string                 Task status
      --tags strings                  Tags to add to the task
      --template string               Format JSON output using a Go template
      --time-estimate string          Time estimate (e.g. 2h, 30m, 1h30m)
      --type int                      Task type (0=task, 1=milestone, or custom type ID) (default -1)
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

