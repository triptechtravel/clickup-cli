---
title: "clickup goal create"
description: "Auto-generated reference for clickup goal create"
---

Create a goal

### Synopsis

Create a new goal in your ClickUp workspace.

```
clickup goal create [flags]
```

### Examples

```
  # Create a goal
  clickup goal create --name "Q1 Revenue Target" --description "Hit $1M ARR"

  # Create with a due date (Unix timestamp in ms)
  clickup goal create --name "Ship v2" --due-date 1704067200000
```

### Options

```
      --color string         Goal color hex
      --description string   Goal description
      --due-date int         Due date (Unix timestamp in ms)
  -h, --help                 help for create
      --name string          Goal name (required)
```

### SEE ALSO

* [clickup goal](/clickup-cli/reference/clickup_goal/)	 - Manage goals

