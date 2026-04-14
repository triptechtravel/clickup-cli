---
title: "clickup space delete"
description: "Auto-generated reference for clickup space delete"
---

Delete a space

### Synopsis

Delete a ClickUp space permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.

```
clickup space delete <space-id> [flags]
```

### Examples

```
  # Delete a space (with confirmation)
  clickup space delete 12345

  # Delete without confirmation
  clickup space delete 12345 --yes
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup space](/clickup-cli/reference/clickup_space/)	 - Manage spaces

