---
title: "clickup doc page view"
description: "Auto-generated reference for clickup doc page view"
---

View a page in a ClickUp Doc

### Synopsis

Display the content and metadata of a specific page within a ClickUp Doc.

```
clickup doc page view <doc-id> <page-id> [flags]
```

### Examples

```
  # View a page
  clickup doc page view abc123 page456

  # View as markdown
  clickup doc page view abc123 page456 --content-format text/md

  # View as JSON
  clickup doc page view abc123 page456 --json
```

### Options

```
      --content-format string   Content format for page body (text/md|text/plain)
  -h, --help                    help for view
      --jq string               Filter JSON output using a jq expression
      --json                    Output JSON
  -r, --raw                     Output raw strings instead of JSON-encoded (use with --jq)
      --template string         Format JSON output using a Go template
```

### SEE ALSO

* [clickup doc page](/clickup-cli/reference/clickup_doc_page/)	 - Manage pages within a ClickUp Doc

