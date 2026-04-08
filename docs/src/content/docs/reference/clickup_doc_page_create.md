---
title: "clickup doc page create"
description: "Auto-generated reference for clickup doc page create"
---

Create a page in a ClickUp Doc

### Synopsis

Create a new page within a ClickUp Doc.

Pages can be nested under other pages using --parent-page-id.
Content can be provided as plain text or markdown.

```
clickup doc page create <doc-id> [flags]
```

### Examples

```
  # Create a basic page
  clickup doc page create abc123 --name "Introduction"

  # Create a page with markdown content
  clickup doc page create abc123 --name "Setup Guide" \
    --content "# Setup\n\nFollow these steps..." --content-format text/md

  # Create a nested page
  clickup doc page create abc123 --name "Advanced Config" \
    --parent-page-id page456

  # Create and output JSON
  clickup doc page create abc123 --name "Release Notes" --json
```

### Options

```
      --content string          Page content
      --content-format string   Content format (text/md|text/plain)
  -h, --help                    help for create
      --jq string               Filter JSON output using a jq expression
      --json                    Output JSON
      --name string             Page name (required)
      --parent-page-id string   Parent page ID (for nested pages)
  -r, --raw                     Output raw strings instead of JSON-encoded (use with --jq)
      --sub-title string        Page subtitle
      --template string         Format JSON output using a Go template
```

### SEE ALSO

* [clickup doc page](/clickup-cli/reference/clickup_doc_page/)	 - Manage pages within a ClickUp Doc

