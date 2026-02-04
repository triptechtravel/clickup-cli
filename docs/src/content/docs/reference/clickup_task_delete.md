---
title: "clickup task delete"
description: "Auto-generated reference for clickup task delete"
---

## clickup task delete

Delete a task

### Synopsis

Delete a ClickUp task permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.

```
clickup task delete <task-id> [flags]
```

### Examples

```
  # Delete a task (with confirmation)
  clickup task delete 86a3xrwkp

  # Delete without confirmation
  clickup task delete CU-abc123 --yes
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

