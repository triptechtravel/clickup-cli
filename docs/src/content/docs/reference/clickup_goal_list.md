---
title: "clickup goal list"
description: "Auto-generated reference for clickup goal list"
---

List goals in the workspace

### Synopsis

List all goals in your ClickUp workspace.

```
clickup goal list [flags]
```

### Examples

```
  # List goals
  clickup goal list

  # Include completed goals
  clickup goal list --completed

  # Output as JSON
  clickup goal list --json
```

### Options

```
      --completed         Include completed goals
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup goal](/clickup-cli/reference/clickup_goal/)	 - Manage goals

