---
title: "clickup comment delete"
description: "Auto-generated reference for clickup comment delete"
---

## clickup comment delete

Delete a comment

### Synopsis

Delete a comment from a ClickUp task.

COMMENT_ID is required. Find comment IDs with 'clickup comment list TASK_ID --json'.
Use --yes to skip the confirmation prompt.

```
clickup comment delete <COMMENT_ID> [flags]
```

### Examples

```
  # Delete a comment (with confirmation)
  clickup comment delete 90160162431205

  # Delete without confirmation
  clickup comment delete 90160162431205 --yes

  # Find comment IDs first
  clickup comment list 86d1rn980 --json | jq '.[].id'
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup comment](/clickup-cli/reference/clickup_comment/)	 - Manage comments on ClickUp tasks

