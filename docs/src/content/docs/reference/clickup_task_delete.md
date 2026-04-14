---
title: "clickup task delete"
description: "Auto-generated reference for clickup task delete"
---

Delete one or more tasks

### Synopsis

Delete one or more ClickUp tasks permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.
Multiple task IDs can be provided for bulk deletion.

```
clickup task delete <task-id> [<task-id>...] [flags]
```

### Examples

```
  # Delete a task (with confirmation)
  clickup task delete 86a3xrwkp

  # Delete without confirmation
  clickup task delete CU-abc123 --yes

  # Bulk delete multiple tasks
  clickup task delete 86abc1 86abc2 86abc3 -y
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

