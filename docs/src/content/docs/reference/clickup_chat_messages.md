---
title: "clickup chat messages"
description: "Auto-generated reference for clickup chat messages"
---

List messages in a Chat channel

### Synopsis

List messages in a ClickUp Chat channel.

```
clickup chat messages <channel-id> [flags]
```

### Examples

```
  # List messages in a channel
  clickup chat messages abc123

  # List messages as JSON
  clickup chat messages abc123 --json
```

### Options

```
  -h, --help              help for messages
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup chat](/clickup-cli/reference/clickup_chat/)	 - Manage ClickUp Chat messages

