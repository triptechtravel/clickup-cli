---
title: "clickup comment add"
description: "Auto-generated reference for clickup comment add"
---

## clickup comment add

Add a comment to a task

### Synopsis

Add a comment to a ClickUp task.

If TASK is not provided, the task ID is auto-detected from the current git branch.
If BODY is not provided (or --editor is used), your editor opens for composing the comment.

```
clickup comment add [TASK] [BODY] [flags]
```

### Examples

```
  # Add a comment to the task detected from the current branch
  clickup comment add "" "Fixed the login bug"

  # Add a comment to a specific task
  clickup comment add abc123 "Deployed to staging"

  # Open your editor to compose the comment
  clickup comment add --editor
```

### Options

```
  -e, --editor   Open editor to compose comment body
  -h, --help     help for add
```

### SEE ALSO

* [clickup comment](/clickup-cli/reference/clickup_comment/)	 - Manage comments on ClickUp tasks

