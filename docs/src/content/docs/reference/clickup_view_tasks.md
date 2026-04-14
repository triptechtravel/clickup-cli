---
title: "clickup view tasks"
description: "Auto-generated reference for clickup view tasks"
---

List tasks in a view

### Synopsis

List all tasks visible in a ClickUp view.

```
clickup view tasks <view-id> [flags]
```

### Examples

```
  # List tasks in a view
  clickup view tasks 3v-abc123

  # Page through results
  clickup view tasks 3v-abc123 --page 1

  # Output as JSON
  clickup view tasks 3v-abc123 --json
```

### Options

```
  -h, --help              help for tasks
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --page int          Page number (0-indexed)
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup view](/clickup-cli/reference/clickup_view/)	 - Manage views

