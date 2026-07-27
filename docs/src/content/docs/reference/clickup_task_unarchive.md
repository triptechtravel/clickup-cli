---
title: "clickup task unarchive"
description: "Auto-generated reference for clickup task unarchive"
---

Restore one or more archived tasks

### Synopsis

Restore archived tasks so they appear in list views again.

As with archiving, subtasks are not included unless --cascade is given.

```
clickup task unarchive <task-id>... [flags]
```

### Examples

```
  # Restore a task
  clickup task unarchive 86abc123

  # Restore a parent and all its subtasks
  clickup task unarchive 86abc123 --cascade
```

### Options

```
      --cascade   Also restore every subtask of the named tasks
  -h, --help      help for unarchive
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

