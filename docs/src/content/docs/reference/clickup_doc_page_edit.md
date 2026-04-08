---
title: "clickup doc page edit"
description: "Auto-generated reference for clickup doc page edit"
---

Edit a page in a ClickUp Doc

### Synopsis

Update the name, subtitle, or content of a page in a ClickUp Doc.

Use --content-edit-mode to control whether content replaces, appends to, or
prepends to the existing page content.

```
clickup doc page edit <doc-id> <page-id> [flags]
```

### Examples

```
  # Replace page content
  clickup doc page edit abc123 page456 --content "# New content"

  # Append to existing content
  clickup doc page edit abc123 page456 \
    --content "## Release Notes\n\n- Fixed bug X" \
    --content-edit-mode append

  # Rename a page
  clickup doc page edit abc123 page456 --name "Updated Title"

  # Edit and output JSON
  clickup doc page edit abc123 page456 --content "Updated" --json
```

### Options

```
      --content string             Page content
      --content-edit-mode string   How to apply content (replace|append|prepend) (default "replace")
      --content-format string      Content format (text/md|text/plain)
  -h, --help                       help for edit
      --jq string                  Filter JSON output using a jq expression
      --json                       Output JSON
      --name string                New page name
  -r, --raw                        Output raw strings instead of JSON-encoded (use with --jq)
      --sub-title string           New page subtitle
      --template string            Format JSON output using a Go template
```

### SEE ALSO

* [clickup doc page](/clickup-cli/reference/clickup_doc_page/)	 - Manage pages within a ClickUp Doc

