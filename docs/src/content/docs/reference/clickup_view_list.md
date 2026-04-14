---
title: "clickup view list"
description: "Auto-generated reference for clickup view list"
---

List views

### Synopsis

List views at different levels of the ClickUp hierarchy.

Specify one of --space, --folder, --list, or --team to choose the scope.
Defaults to --team (workspace-level views) if none specified.

```
clickup view list [flags]
```

### Examples

```
  # List workspace-level views
  clickup view list --team

  # List views in a space
  clickup view list --space 12345

  # List views in a folder
  clickup view list --folder 67890

  # List views in a list
  clickup view list --list abc123
```

### Options

```
      --folder string     List views in a folder
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --list string       List views in a list
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --space string      List views in a space
      --team              List workspace-level views (default)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup view](/clickup-cli/reference/clickup_view/)	 - Manage views

