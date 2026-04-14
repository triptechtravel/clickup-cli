---
title: "clickup task move"
description: "Auto-generated reference for clickup task move"
---

Move a task to a different list

### Synopsis

Move a ClickUp task to a different list.

The task's home list is changed to the target list. Use --move-custom-fields
to carry custom field values from the current list to the new list.

```
clickup task move <task-id> [flags]
```

### Examples

```
  # Move a task to a different list
  clickup task move 86abc123 --list 901613544162

  # Move and carry custom fields
  clickup task move 86abc123 --list 901613544162 --move-custom-fields

  # Auto-detect task from branch
  clickup task move --list 901613544162
```

### Options

```
  -h, --help                 help for move
      --jq string            Filter JSON output using a jq expression
      --json                 Output JSON
      --list string          Target list ID (required)
      --move-custom-fields   Carry custom fields to the new list
  -r, --raw                  Output raw strings instead of JSON-encoded (use with --jq)
      --template string      Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

