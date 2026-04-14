---
title: "clickup chat delete"
description: "Auto-generated reference for clickup chat delete"
---

Delete a Chat message

### Synopsis

Delete a message from a ClickUp Chat channel.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.

```
clickup chat delete <message-id> [flags]
```

### Examples

```
  # Delete a message (with confirmation)
  clickup chat delete msg123

  # Delete without confirmation
  clickup chat delete msg123 --yes
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup chat](/clickup-cli/reference/clickup_chat/)	 - Manage ClickUp Chat messages

