---
title: "clickup goal delete"
description: "Auto-generated reference for clickup goal delete"
---

Delete a goal

### Synopsis

Delete a ClickUp goal permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.

```
clickup goal delete <goal-id> [flags]
```

### Examples

```
  # Delete a goal (with confirmation)
  clickup goal delete e53a33d0-2eb2-4664-a4b3-5e1b0df0e912

  # Delete without confirmation
  clickup goal delete e53a33d0-2eb2-4664-a4b3-5e1b0df0e912 --yes
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup goal](/clickup-cli/reference/clickup_goal/)	 - Manage goals

