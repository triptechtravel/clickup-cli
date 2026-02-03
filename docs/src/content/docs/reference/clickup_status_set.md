---
title: "clickup status set"
description: "Auto-generated reference for clickup status set"
---

## clickup status set

Set the status of a task

### Synopsis

Change a task's status using fuzzy matching.

The STATUS argument is matched against available statuses for the task's space.
Matching priority: exact match, then case-insensitive contains, then fuzzy match.

If TASK is not provided, the task ID is auto-detected from the current git branch.

```
clickup status set <status> [task] [flags]
```

### Examples

```
  # Set status using auto-detected task from branch
  clickup status set "in progress"

  # Set status for a specific task
  clickup status set "done" CU-abc123

  # Fuzzy matching works too
  clickup status set "prog" CU-abc123
```

### Options

```
  -h, --help   help for set
```

### SEE ALSO

* [clickup status](/clickup-cli/reference/clickup_status/)	 - Manage task statuses

