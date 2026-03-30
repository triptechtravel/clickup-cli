---
title: "clickup comment reply"
description: "Auto-generated reference for clickup comment reply"
---

Reply to a comment thread

### Synopsis

Reply to an existing comment on a ClickUp task, creating a threaded reply.

Use 'clickup comment list <task> --json' to find comment IDs.
Use @username in the body to @mention workspace members.

```
clickup comment reply <comment-id> [BODY] [flags]
```

### Examples

```
  # Reply to a specific comment
  clickup comment reply 90160175975219 "Yes, that's confirmed"

  # Reply with @mentions
  clickup comment reply 90160175975219 "@Michelle confirmed, BookEasy only"

  # Open editor to compose the reply
  clickup comment reply 90160175975219 --editor
```

### Options

```
  -e, --editor   Open editor to compose reply body
  -h, --help     help for reply
```

### SEE ALSO

* [clickup comment](/clickup-cli/reference/clickup_comment/)	 - Manage comments on ClickUp tasks

