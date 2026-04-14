---
title: "clickup folder delete"
description: "Auto-generated reference for clickup folder delete"
---

Delete a folder

### Synopsis

Delete a ClickUp folder permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.

```
clickup folder delete <folder-id> [flags]
```

### Examples

```
  # Delete a folder (with confirmation)
  clickup folder delete 12345

  # Delete without confirmation
  clickup folder delete 12345 --yes
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup folder](/clickup-cli/reference/clickup_folder/)	 - Manage folders

