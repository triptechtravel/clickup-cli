---
title: "clickup task checklist item edit"
description: "Auto-generated reference for clickup task checklist item edit"
---

Edit one or more checklist items (rename or assign)

```
clickup task checklist item edit <checklist-id> <item-id> [<item-id>...] [flags]
```

### Examples

```
  # Assign a checklist item to a user
  clickup task checklist item edit b955c4dc-example 21e08dc8-example --assignee 54874661

  # Bulk assign all items in a checklist
  clickup task checklist item edit b955c4dc-example item1 item2 item3 --assignee 54874661

  # Rename a checklist item
  clickup task checklist item edit b955c4dc-example 21e08dc8-example --name "Updated name"

  # Unassign (set assignee to 0)
  clickup task checklist item edit b955c4dc-example 21e08dc8-example --assignee 0
```

### Options

```
      --assignee int   User ID to assign the item to (0 to unassign; see 'clickup member list') (default -1)
  -h, --help           help for edit
      --name string    New name for the item (single item only)
```

### SEE ALSO

* [clickup task checklist item](/clickup-cli/reference/clickup_task_checklist_item/)	 - Manage checklist items

