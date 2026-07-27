---
title: "clickup task archive"
description: "Auto-generated reference for clickup task archive"
---

Archive one or more tasks

### Synopsis

Archive tasks so they no longer appear in list views.

Archiving is reversible with 'clickup task unarchive'.

Subtasks are NOT archived with their parent. ClickUp does not cascade, so
archiving a parent silently leaves its children active. This command reports
any subtasks it left behind; pass --cascade to include them.

```
clickup task archive <task-id>... [flags]
```

### Examples

```
  # Archive a task
  clickup task archive 86abc123

  # Archive several at once
  clickup task archive 86abc1 86abc2 86abc3

  # Archive a parent and all its subtasks
  clickup task archive 86abc123 --cascade
```

### Options

```
      --cascade   Also archive every subtask of the named tasks
  -h, --help      help for archive
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

