---
title: "clickup template list"
description: "Auto-generated reference for clickup template list"
---

List templates

### Synopsis

List templates available in your ClickUp workspace.

Use --type to filter by template type: task (default), folder, or list.

```
clickup template list [flags]
```

### Examples

```
  # List task templates
  clickup template list

  # List folder templates
  clickup template list --type folder

  # List list templates
  clickup template list --type list

  # Output as JSON
  clickup template list --json
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --template string   Format JSON output using a Go template
      --type string       Template type: task, folder, or list (default "task")
```

### SEE ALSO

* [clickup template](/clickup-cli/reference/clickup_template/)	 - Manage templates

