---
title: "clickup doc list"
description: "Auto-generated reference for clickup doc list"
---

List ClickUp Docs in the workspace

### Synopsis

List Docs in the configured ClickUp workspace.

Supports filtering by creator, status, parent location, and pagination.

```
clickup doc list [flags]
```

### Examples

```
  # List all Docs
  clickup doc list

  # List non-deleted, non-archived Docs in JSON
  clickup doc list --json

  # List Docs in a specific space
  clickup doc list --parent-id 123456 --parent-type SPACE

  # Paginate
  clickup doc list --limit 10 --cursor <cursor>
```

### Options

```
      --archived             Include archived Docs
      --creator int          Filter by creator user ID
      --cursor string        Pagination cursor from a previous response
      --deleted              Include deleted Docs
  -h, --help                 help for list
      --jq string            Filter JSON output using a jq expression
      --json                 Output JSON
      --limit int            Maximum number of Docs to return
      --parent-id string     Filter by parent ID
      --parent-type string   Parent type (SPACE|FOLDER|LIST|WORKSPACE|EVERYTHING)
      --template string      Format JSON output using a Go template
```

### SEE ALSO

* [clickup doc](/clickup-cli/reference/clickup_doc/)	 - Manage ClickUp Docs

