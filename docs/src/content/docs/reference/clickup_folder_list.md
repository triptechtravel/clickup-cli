---
title: "clickup folder list"
description: "Auto-generated reference for clickup folder list"
---

List folders in a space

### Synopsis

List all folders in a ClickUp space.

```
clickup folder list [flags]
```

### Examples

```
  # List folders in your default space
  clickup folder list

  # List folders in a specific space
  clickup folder list --space 12345

  # Include archived folders
  clickup folder list --archived
```

### Options

```
      --archived          Include archived folders
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --space string      Space ID (defaults to configured space)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup folder](/clickup-cli/reference/clickup_folder/)	 - Manage folders

