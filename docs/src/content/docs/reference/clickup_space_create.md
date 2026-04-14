---
title: "clickup space create"
description: "Auto-generated reference for clickup space create"
---

Create a new space

### Synopsis

Create a new ClickUp space in your workspace.

The --name flag is required. If --team is not provided, the configured
workspace is used.

```
clickup space create [flags]
```

### Examples

```
  # Create a space
  clickup space create --name "Dev"

  # Create in a specific workspace
  clickup space create --name "Dev" --team 12345
```

### Options

```
  -h, --help              help for create
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --name string       Space name (required)
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --team string       Workspace/team ID (defaults to configured workspace)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup space](/clickup-cli/reference/clickup_space/)	 - Manage spaces

