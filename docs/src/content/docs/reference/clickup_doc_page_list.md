---
title: "clickup doc page list"
description: "Auto-generated reference for clickup doc page list"
---

List pages in a ClickUp Doc

### Synopsis

List all pages in a ClickUp Doc. Pages are returned as a tree structure; use --max-depth to control nesting depth.

```
clickup doc page list <doc-id> [flags]
```

### Examples

```
  # List all pages in a Doc
  clickup doc page list abc123

  # List top-level pages only
  clickup doc page list abc123 --max-depth 0

  # List pages as JSON
  clickup doc page list abc123 --json
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --max-depth float   Maximum page nesting depth (-1 for unlimited) (default -1)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup doc page](/clickup-cli/reference/clickup_doc_page/)	 - Manage pages within a ClickUp Doc

