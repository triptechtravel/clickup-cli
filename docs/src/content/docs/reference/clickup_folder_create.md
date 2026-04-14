---
title: "clickup folder create"
description: "Auto-generated reference for clickup folder create"
---

Create a new folder

### Synopsis

Create a new ClickUp folder in a space.

The --name flag is required. If --space is not provided, the configured
space is used.

```
clickup folder create [flags]
```

### Examples

```
  # Create a folder in the current space
  clickup folder create --name "Sprint Folder"

  # Create in a specific space
  clickup folder create --name "Sprint Folder" --space 67890
```

### Options

```
  -h, --help              help for create
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --name string       Folder name (required)
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --space string      Space ID (defaults to configured space)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup folder](/clickup-cli/reference/clickup_folder/)	 - Manage folders

