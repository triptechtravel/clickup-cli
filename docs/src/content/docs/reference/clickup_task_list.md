---
title: "clickup task list"
description: "Auto-generated reference for clickup task list"
---

## clickup task list

List tasks in a ClickUp list

### Synopsis

List tasks from a ClickUp list with optional filters.

The --list-id flag is required to specify which ClickUp list to query.
Results can be filtered by assignee, status, and sprint.

```
clickup task list [flags]
```

### Examples

```
  # List tasks in a ClickUp list
  clickup task list --list-id 12345

  # Filter by assignee and status
  clickup task list --list-id 12345 --assignee me --status "in progress"
```

### Options

```
      --assignee strings   Filter by assignee ID(s), or "me" for yourself
  -h, --help               help for list
      --jq string          Filter JSON output using a jq expression
      --json               Output JSON
      --list-id string     ClickUp list ID (required)
      --page int           Page number for pagination (starts at 0)
      --sprint string      Filter by sprint name
      --status strings     Filter by status(es)
      --template string    Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

