---
title: "clickup task dependency remove"
description: "Auto-generated reference for clickup task dependency remove"
---

## clickup task dependency remove

Remove a dependency from a task

### Synopsis

Remove a dependency relationship between two tasks.

Use --depends-on to remove a "waits for" relationship.
Use --blocks to remove a "blocks" relationship.

```
clickup task dependency remove <task-id> [flags]
```

### Examples

```
  # Remove depends-on relationship
  clickup task dependency remove 86abc123 --depends-on 86def456

  # Remove blocks relationship
  clickup task dependency remove 86abc123 --blocks 86def456
```

### Options

```
      --blocks string       Task ID to remove blocks relationship with
      --depends-on string   Task ID to remove depends-on relationship with
  -h, --help                help for remove
```

### SEE ALSO

* [clickup task dependency](/clickup-cli/reference/clickup_task_dependency/)	 - Manage task dependencies

