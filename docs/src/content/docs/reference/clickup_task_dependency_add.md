---
title: "clickup task dependency add"
description: "Auto-generated reference for clickup task dependency add"
---

## clickup task dependency add

Add a dependency to a task

### Synopsis

Add a dependency relationship between two tasks.

Use --depends-on to indicate this task waits on another task.
Use --blocks to indicate this task blocks another task.

```
clickup task dependency add <task-id> [flags]
```

### Examples

```
  # This task depends on (waits for) another task
  clickup task dependency add 86abc123 --depends-on 86def456

  # This task blocks another task
  clickup task dependency add 86abc123 --blocks 86def456
```

### Options

```
      --blocks string       Task ID that this task blocks
      --depends-on string   Task ID that this task depends on (waits for)
  -h, --help                help for add
```

### SEE ALSO

* [clickup task dependency](/clickup-cli/reference/clickup_task_dependency/)	 - Manage task dependencies

