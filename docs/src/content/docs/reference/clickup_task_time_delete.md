---
title: "clickup task time delete"
description: "Auto-generated reference for clickup task time delete"
---

## clickup task time delete

Delete a time entry

### Synopsis

Delete a time entry from ClickUp.

ENTRY_ID is required. Find entry IDs with 'clickup task time list TASK_ID'.
Use --yes to skip the confirmation prompt.

```
clickup task time delete <entry-id> [flags]
```

### Examples

```
  # Delete a time entry (with confirmation)
  clickup task time delete 1234567890

  # Delete without confirmation
  clickup task time delete 1234567890 --yes

  # Find entry IDs first
  clickup task time list 86a3xrwkp
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup task time](/clickup-cli/reference/clickup_task_time/)	 - Track time on ClickUp tasks

