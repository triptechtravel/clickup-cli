---
title: "clickup goal view"
description: "Auto-generated reference for clickup goal view"
---

View a goal

### Synopsis

View detailed information about a ClickUp goal.

```
clickup goal view <goal-id> [flags]
```

### Examples

```
  # View a goal
  clickup goal view e53a33d0-2eb2-4664-a4b3-5e1b0df0e912

  # View as JSON
  clickup goal view e53a33d0-2eb2-4664-a4b3-5e1b0df0e912 --json
```

### Options

```
  -h, --help              help for view
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup goal](/clickup-cli/reference/clickup_goal/)	 - Manage goals

