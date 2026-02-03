---
title: "clickup link"
description: "Auto-generated reference for clickup link"
---

## clickup link

Link GitHub objects to ClickUp tasks

### Synopsis

Link pull requests, branches, and commits to ClickUp tasks.

By default, links are stored in a managed section of the task description
using ClickUp's markdown_description API field, so they render as rich text
with clickable links, bold formatting, and code blocks directly in the
ClickUp UI. Running the same command again updates the existing entry rather
than creating duplicates.

Optionally, configure a custom field for link storage:
  Set 'link_field' in ~/.config/clickup/config.yml to the name of a
  URL or text custom field in your workspace (e.g., link_field: "link_url").
  Per-directory overrides are also supported in directory_defaults.

### Options

```
  -h, --help   help for link
```

### SEE ALSO

* [clickup](/clickup-cli/reference/clickup/)	 - ClickUp CLI - manage tasks from the command line
* [clickup link branch](/clickup-cli/reference/clickup_link_branch/)	 - Link the current git branch to a ClickUp task
* [clickup link commit](/clickup-cli/reference/clickup_link_commit/)	 - Link a git commit to a ClickUp task
* [clickup link pr](/clickup-cli/reference/clickup_link_pr/)	 - Link a GitHub PR to a ClickUp task
* [clickup link sync](/clickup-cli/reference/clickup_link_sync/)	 - Sync ClickUp task info to a GitHub PR

