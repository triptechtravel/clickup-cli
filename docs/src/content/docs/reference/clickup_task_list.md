---
title: "clickup task list"
description: "Auto-generated reference for clickup task list"
---

List tasks in a ClickUp list

### Synopsis

List tasks from a ClickUp list with optional filters.

If --list-id is not provided, the configured default list is used
(set via 'clickup list select'). Results can be filtered by assignee,
status, and sprint.

```
clickup task list [flags]
```

### Examples

```
  # List tasks using your configured default list
  clickup task list

  # List tasks in a specific ClickUp list
  clickup task list --list-id 12345

  # Filter by assignee and status
  clickup task list --list-id 12345 --assignee me --status "in progress"

  # Include closed tasks
  clickup task list --list-id 12345 --include-closed
```

### Options

```
      --assignee strings   Filter by assignee ID(s), or "me" for yourself
  -h, --help               help for list
  -c, --include-closed     Include closed/completed tasks
      --jq string          Filter JSON output using a jq expression
      --json               Output JSON
      --list-id string     ClickUp list ID (defaults to configured list)
      --page int           Page number for pagination (starts at 0)
  -r, --raw                Output raw strings instead of JSON-encoded (use with --jq)
      --sprint string      Filter by sprint name
      --status strings     Filter by status(es)
      --template string    Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

