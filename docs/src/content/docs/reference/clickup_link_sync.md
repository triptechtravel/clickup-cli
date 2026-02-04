---
title: "clickup link sync"
description: "Auto-generated reference for clickup link sync"
---

## clickup link sync

Sync ClickUp task info to a GitHub PR

### Synopsis

Update a GitHub pull request with information from the linked ClickUp task.

Adds the ClickUp task URL and status to the PR body, and updates the task
description (or configured custom field) with a link to the PR.

The task ID is auto-detected from the branch name, or specified with --task.
When --task is specified and no PR is found for the current branch, the CLI
searches for PRs whose branch name contains the task ID (useful after merging).

```
clickup link sync [PR-NUMBER] [flags]
```

### Examples

```
  # Sync current branch's PR with the detected task
  clickup link sync

  # Sync a specific PR with a specific task
  clickup link sync 1109 --repo owner/repo --task 86d1rn980
```

### Options

```
  -h, --help          help for sync
      --repo string   GitHub repository (owner/repo)
      --task string   ClickUp task ID (auto-detected from branch if not set)
```

### SEE ALSO

* [clickup link](/clickup-cli/reference/clickup_link/)	 - Link GitHub objects to ClickUp tasks

