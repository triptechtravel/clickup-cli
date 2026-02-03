---
title: "clickup field list"
description: "Auto-generated reference for clickup field list"
---

## clickup field list

List custom fields available in a list

### Synopsis

Display all custom fields accessible in the specified ClickUp list.

Shows the field name, type, field ID (needed for API calls), and any
available options for dropdown or label fields.

```
clickup field list [flags]
```

### Examples

```
  # List custom fields for a specific list
  clickup field list --list-id 901234567

  # Output as JSON
  clickup field list --list-id 901234567 --json
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --list-id string    ClickUp list ID (required)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup field](/clickup-cli/reference/clickup_field/)	 - Manage custom fields

