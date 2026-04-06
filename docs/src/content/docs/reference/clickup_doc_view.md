---
title: "clickup doc view"
description: "Auto-generated reference for clickup doc view"
---

View a ClickUp Doc

### Synopsis

Display details about a ClickUp Doc including its metadata and parent location.

```
clickup doc view <doc-id> [flags]
```

### Examples

```
  # View a Doc
  clickup doc view abc123

  # View as JSON
  clickup doc view abc123 --json
```

### Options

```
  -h, --help              help for view
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup doc](/clickup-cli/reference/clickup_doc/)	 - Manage ClickUp Docs

