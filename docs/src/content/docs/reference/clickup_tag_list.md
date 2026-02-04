---
title: "clickup tag list"
description: "Auto-generated reference for clickup tag list"
---

## clickup tag list

List tags in a space

### Synopsis

Display all tags available in a ClickUp space.

Uses the default space from your config unless --space-id is provided.

```
clickup tag list [flags]
```

### Examples

```
  # List tags for the default space
  clickup tag list

  # List tags for a specific space
  clickup tag list --space-id 12345678

  # Output as JSON
  clickup tag list --json
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --space-id string   Space ID (defaults to configured space)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup tag](/clickup-cli/reference/clickup_tag/)	 - Manage space tags

