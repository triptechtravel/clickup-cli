---
title: "clickup webhook delete"
description: "Auto-generated reference for clickup webhook delete"
---

Delete a webhook

### Synopsis

Delete a ClickUp webhook permanently.

This action cannot be undone. A confirmation prompt is shown unless --yes is passed.

```
clickup webhook delete <webhook-id> [flags]
```

### Examples

```
  # Delete a webhook (with confirmation)
  clickup webhook delete 4b67ac88

  # Delete without confirmation
  clickup webhook delete 4b67ac88 --yes
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip confirmation prompt
```

### SEE ALSO

* [clickup webhook](/clickup-cli/reference/clickup_webhook/)	 - Manage webhooks

