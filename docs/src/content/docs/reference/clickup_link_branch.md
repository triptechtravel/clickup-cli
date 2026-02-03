---
title: "clickup link branch"
description: "Auto-generated reference for clickup link branch"
---

## clickup link branch

Link the current git branch to a ClickUp task

### Synopsis

Link the current git branch to a ClickUp task.

Updates the task description (or a configured custom field) with a reference
to the branch. Running the command again updates the existing entry rather
than creating duplicates.

The ClickUp task ID is auto-detected from the current git branch name,
or can be specified explicitly with --task.

```
clickup link branch [flags]
```

### Examples

```
  # Link current branch to auto-detected task
  clickup link branch

  # Link to a specific task
  clickup link branch --task CU-abc123
```

### Options

```
  -h, --help          help for branch
      --task string   ClickUp task ID (auto-detected from branch if not set)
```

### SEE ALSO

* [clickup link](/clickup-cli/reference/clickup_link/)	 - Link GitHub objects to ClickUp tasks

