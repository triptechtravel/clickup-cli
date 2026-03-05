---
title: "clickup status add"
description: "Auto-generated reference for clickup status add"
---

Add a new status to a space

### Synopsis

Add a new custom status to a ClickUp space.

The new status is inserted before the final "Closed"/"done" status in the
workflow ordering. Since statuses affect all tasks in a space, this command
requires interactive confirmation unless -y is passed.

```
clickup status add <name> [flags]
```

### Examples

```
  # Add a "done" status to the default space
  clickup status add "done"

  # Add with a specific color
  clickup status add "QA Review" --color "#7C4DFF"

  # Skip confirmation prompt
  clickup status add "done" -y

  # Add to a specific space
  clickup status add "done" --space 12345
```

### Options

```
      --color string   Status color hex (e.g. "#7C4DFF"); omit to let ClickUp pick
  -h, --help           help for add
      --space string   Space ID (defaults to configured space)
  -y, --yes            Skip confirmation prompt
```

### SEE ALSO

* [clickup status](/clickup-cli/reference/clickup_status/)	 - Manage task statuses

