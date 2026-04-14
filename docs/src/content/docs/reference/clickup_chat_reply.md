---
title: "clickup chat reply"
description: "Auto-generated reference for clickup chat reply"
---

Reply to a Chat message

### Synopsis

Reply to a message in a ClickUp Chat channel.

```
clickup chat reply <message-id> <text> [flags]
```

### Examples

```
  # Reply to a message
  clickup chat reply msg123 "Got it, thanks!"

  # Reply and get JSON response
  clickup chat reply msg123 "On it" --json
```

### Options

```
  -h, --help              help for reply
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup chat](/clickup-cli/reference/clickup_chat/)	 - Manage ClickUp Chat messages

