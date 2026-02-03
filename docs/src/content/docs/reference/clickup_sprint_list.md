---
title: "clickup sprint list"
description: "Auto-generated reference for clickup sprint list"
---

## clickup sprint list

List sprints in a folder

### Synopsis

List sprints (lists) within a sprint folder.

Sprints in ClickUp are organized as lists within a folder.
This command finds sprint folders in your configured space
and lists the sprints within them.

If multiple sprint folders exist, use --folder to specify one,
or the CLI will remember your choice.

```
clickup sprint list [flags]
```

### Examples

```
  # List all sprints
  clickup sprint list

  # Specify sprint folder
  clickup sprint list --folder 132693664
```

### Options

```
      --folder string     Sprint folder ID (auto-detected if not set)
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup sprint](/clickup-cli/reference/clickup_sprint/)	 - Manage sprints

