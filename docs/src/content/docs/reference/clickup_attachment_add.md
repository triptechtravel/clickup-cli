---
title: "clickup attachment add"
description: "Auto-generated reference for clickup attachment add"
---

Upload file(s) to a task

### Synopsis

Upload one or more files as attachments to a ClickUp task.

If TASK is not provided, the task ID is auto-detected from the current git branch.
When the first argument is a file that exists on disk, all arguments are treated
as files and the task is auto-detected.

```
clickup attachment add [TASK] <FILE>... [flags]
```

### Examples

```
  # Upload a file to the task detected from the current branch
  clickup attachment add screenshot.png

  # Upload to a specific task
  clickup attachment add abc123 report.pdf

  # Upload multiple files
  clickup attachment add abc123 file1.png file2.pdf
```

### Options

```
  -h, --help   help for add
```

### SEE ALSO

* [clickup attachment](/clickup-cli/reference/clickup_attachment/)	 - Manage attachments on ClickUp tasks

