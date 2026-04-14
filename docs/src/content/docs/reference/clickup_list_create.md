---
title: "clickup list create"
description: "Auto-generated reference for clickup list create"
---

Create a new list

### Synopsis

Create a new ClickUp list in a folder or space.

Use --folder to create a list inside a folder.
Use --space to create a folderless list directly in a space.
One of --folder or --space is required.

```
clickup list create [flags]
```

### Examples

```
  # Create a list in a folder
  clickup list create --name "Backlog" --folder 12345

  # Create a folderless list in a space
  clickup list create --name "Backlog" --space 67890
```

### Options

```
      --folder string     Folder ID to create the list in
  -h, --help              help for create
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --name string       List name (required)
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --space string      Space ID to create a folderless list in
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup list](/clickup-cli/reference/clickup_list/)	 - Manage lists

