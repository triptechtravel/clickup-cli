---
title: "clickup webhook create"
description: "Auto-generated reference for clickup webhook create"
---

Create a webhook

### Synopsis

Create a new webhook in your ClickUp workspace.

```
clickup webhook create [flags]
```

### Examples

```
  # Create a webhook for all events
  clickup webhook create --endpoint https://example.com/hook --events "*"

  # Create a webhook for specific events
  clickup webhook create --endpoint https://example.com/hook --events taskCreated --events taskUpdated
```

### Options

```
      --endpoint string   Webhook endpoint URL (required)
      --events strings    Event types to subscribe to (required)
  -h, --help              help for create
```

### SEE ALSO

* [clickup webhook](/clickup-cli/reference/clickup_webhook/)	 - Manage webhooks

