---
title: "clickup doc create"
description: "Auto-generated reference for clickup doc create"
---

Create a new ClickUp Doc

### Synopsis

Create a new Doc in the configured ClickUp workspace.

Optionally scope the Doc to a parent space, folder, or list.
The --create-page flag (default true) creates an initial empty page.

```
clickup doc create [flags]
```

### Examples

```
  # Create a Doc with default visibility
  clickup doc create --name "Project Runbook"

  # Create a Doc in a specific space
  clickup doc create --name "Team Wiki" --parent-id 123456 --parent-type SPACE

  # Create a Doc with public visibility and no initial page
  clickup doc create --name "Public Docs" --visibility PUBLIC --create-page=false

  # Create and output JSON
  clickup doc create --name "API Reference" --json
```

### Options

```
      --create-page          Create an initial empty page in the Doc (default true)
  -h, --help                 help for create
      --jq string            Filter JSON output using a jq expression
      --json                 Output JSON
      --name string          Doc name (required)
      --parent-id string     Parent ID (space, folder, or list)
      --parent-type string   Parent type (SPACE|FOLDER|LIST|WORKSPACE|EVERYTHING)
      --template string      Format JSON output using a Go template
      --visibility string    Visibility (PUBLIC|PRIVATE|PERSONAL|HIDDEN)
```

### SEE ALSO

* [clickup doc](/clickup-cli/reference/clickup_doc/)	 - Manage ClickUp Docs

