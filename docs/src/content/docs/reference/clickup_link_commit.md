---
title: "clickup link commit"
description: "Auto-generated reference for clickup link commit"
---

## clickup link commit

Link a git commit to a ClickUp task

### Synopsis

Link a git commit to a ClickUp task.

Updates the task description (or a configured custom field) with a link to
the commit. Running the command again updates the existing entry rather than
creating duplicates.

If SHA is not provided, the HEAD commit is used.
The ClickUp task ID is auto-detected from the current git branch name,
or can be specified explicitly with --task.

```
clickup link commit [SHA] [flags]
```

### Examples

```
  # Link the latest commit
  clickup link commit

  # Link a specific commit
  clickup link commit a1b2c3d

  # Link to a specific task and repo
  clickup link commit a1b2c3d --task CU-abc123 --repo owner/repo
```

### Options

```
  -h, --help          help for commit
      --repo string   GitHub repository (owner/repo) for the commit URL
      --task string   ClickUp task ID (auto-detected from branch if not set)
```

### SEE ALSO

* [clickup link](/clickup-cli/reference/clickup_link/)	 - Link GitHub objects to ClickUp tasks

