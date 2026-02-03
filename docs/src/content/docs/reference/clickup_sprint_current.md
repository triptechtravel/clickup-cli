---
title: "clickup sprint current"
description: "Auto-generated reference for clickup sprint current"
---

## clickup sprint current

Show current sprint tasks

### Synopsis

Show tasks in the currently active sprint.

Finds the sprint whose dates contain today, then lists
all tasks grouped by status with assignees, priorities,
and linked GitHub branches.

```
clickup sprint current [flags]
```

### Examples

```
  # Show current sprint tasks
  clickup sprint current

  # Specify a sprint folder
  clickup sprint current --folder 132693664

  # JSON output
  clickup sprint current --json
```

### Options

```
      --folder string     Sprint folder ID (auto-detected if not set)
  -h, --help              help for current
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup sprint](/clickup-cli/reference/clickup_sprint/)	 - Manage sprints

