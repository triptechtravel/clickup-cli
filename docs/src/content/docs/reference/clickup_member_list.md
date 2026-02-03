---
title: "clickup member list"
description: "Auto-generated reference for clickup member list"
---

## clickup member list

List workspace members

### Synopsis

List all members in the configured ClickUp workspace.

Displays each member's ID, username, email, and role. Member IDs
are useful for assigning tasks, adding watchers, and tagging users.

```
clickup member list [flags]
```

### Examples

```
  # List workspace members
  clickup member list

  # JSON output for scripting
  clickup member list --json

  # Get a specific member's ID
  clickup member list --json --jq '.[] | select(.username == "Isaac") | .id'
```

### Options

```
  -h, --help              help for list
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup member](/clickup-cli/reference/clickup_member/)	 - Manage workspace members

