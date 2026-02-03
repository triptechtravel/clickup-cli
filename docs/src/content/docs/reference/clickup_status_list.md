---
title: "clickup status list"
description: "Auto-generated reference for clickup status list"
---

## clickup status list

List available statuses for a space

### Synopsis

List all available statuses configured for a ClickUp space.

Uses the --space flag or falls back to the default space from configuration.

```
clickup status list [flags]
```

### Examples

```
  # List statuses for the default space
  clickup status list

  # List statuses for a specific space
  clickup status list --space 12345678

  # Output as JSON
  clickup status list --json
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --space string      Space ID (defaults to configured space)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup status](/clickup-cli/reference/clickup_status/)	 - Manage task statuses

