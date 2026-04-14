---
title: "clickup task checklist item add"
description: "Auto-generated reference for clickup task checklist item add"
---

Add an item to a checklist

```
clickup task checklist item add <checklist-id> <item-name> [flags]
```

### Examples

```
  # Add an item
  clickup task checklist item add b955c4dc-example "Run migrations"

  # Add an item assigned to a user (use 'clickup member list' to find IDs)
  clickup task checklist item add b955c4dc-example "Run migrations" --assignee 54874661
```

### Options

```
      --assignee int   User ID to assign the item to (see 'clickup member list')
  -h, --help           help for add
```

### SEE ALSO

* [clickup task checklist item](/clickup-cli/reference/clickup_task_checklist_item/)	 - Manage checklist items

