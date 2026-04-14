---
title: "clickup template use"
description: "Auto-generated reference for clickup template use"
---

Create a task from a template

### Synopsis

Create a new task from an existing task template.

```
clickup template use <template-id> [flags]
```

### Examples

```
  # Create a task from a template
  clickup template use t-12345 --list 67890 --name "New Task from Template"
```

### Options

```
  -h, --help          help for use
      --list string   List ID to create the task in (required)
      --name string   Name for the new task (required)
```

### SEE ALSO

* [clickup template](/clickup-cli/reference/clickup_template/)	 - Manage templates

