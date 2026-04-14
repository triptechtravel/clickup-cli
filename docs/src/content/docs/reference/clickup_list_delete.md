---
title: "clickup list delete"
description: "Auto-generated reference for clickup list delete"
---

Delete a list

### Synopsis

Delete a ClickUp list permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.

```
clickup list delete <list-id> [flags]
```

### Examples

```
  # Delete a list (with confirmation)
  clickup list delete 12345

  # Delete without confirmation
  clickup list delete 12345 --yes
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup list](/clickup-cli/reference/clickup_list/)	 - Manage lists

