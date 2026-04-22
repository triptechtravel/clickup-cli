---
title: "clickup view get"
description: "Auto-generated reference for clickup view get"
---

Get a view

### Synopsis

Get detailed information about a ClickUp view.

Output is always JSON because the response is a union type that cannot
be rendered as a table.

```
clickup view get <view-id> [flags]
```

### Examples

```
  # Get a view
  clickup view get 3v-abc123
```

### Options

```
  -h, --help              help for get
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup view](/clickup-cli/reference/clickup_view/)	 - Manage views

