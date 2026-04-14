---
title: "clickup list list"
description: "Auto-generated reference for clickup list list"
---

List ClickUp lists in a folder or space

### Synopsis

List ClickUp lists. If --folder is provided, lists within that folder
are returned. If only --space is provided, folderless lists in that space
are returned.

```
clickup list list [flags]
```

### Examples

```
  # List lists in your default folder
  clickup list list

  # List lists in a specific folder
  clickup list list --folder 12345

  # List folderless lists in a space
  clickup list list --space 67890
```

### Options

```
      --archived          Include archived lists
      --folder string     Folder ID (defaults to configured folder)
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --space string      Space ID (defaults to configured space, used for folderless lists)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup list](/clickup-cli/reference/clickup_list/)	 - Manage lists

