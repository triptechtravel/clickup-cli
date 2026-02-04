---
title: "clickup link pr"
description: "Auto-generated reference for clickup link pr"
---

## clickup link pr

Link a GitHub PR to a ClickUp task

### Synopsis

Link a GitHub pull request to a ClickUp task.

Updates the task description (or a configured custom field) with a link to
the PR. Running the command again updates the existing entry rather than
creating duplicates.

If NUMBER is not provided, the current PR is detected using the GitHub CLI (gh).
When --task is specified and no PR is found for the current branch, the CLI
searches for PRs whose branch name contains the task ID (useful after merging).
The ClickUp task ID is auto-detected from the current git branch name,
or can be specified explicitly with --task.

```
clickup link pr [NUMBER] [flags]
```

### Examples

```
  # Link the current branch's PR to the detected task
  clickup link pr

  # Link a specific PR by number
  clickup link pr 42

  # Link a PR from another repo to a specific task
  clickup link pr 1109 --repo owner/repo --task 86d1rn980
```

### Options

```
  -h, --help          help for pr
      --repo string   GitHub repository (owner/repo) for the PR
      --task string   ClickUp task ID (auto-detected from branch if not set)
```

### SEE ALSO

* [clickup link](/clickup-cli/reference/clickup_link/)	 - Link GitHub objects to ClickUp tasks

