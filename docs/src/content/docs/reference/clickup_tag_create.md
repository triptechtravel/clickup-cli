---
title: "clickup tag create"
description: "Auto-generated reference for clickup tag create"
---

Create tags in a space

### Synopsis

Create one or more tags in a ClickUp space.

If a tag already exists, it is skipped with a message. Uses the default space
from your config unless --space-id is provided.

```
clickup tag create <name> [<name>...] [flags]
```

### Examples

```
  # Create a single tag
  clickup tag create feat:search

  # Create multiple tags at once
  clickup tag create feat:search feat:maps fix:auth

  # Create in a specific space
  clickup tag create my-tag --space-id 12345678
```

### Options

```
  -h, --help              help for create
      --space-id string   Space ID (defaults to configured space)
```

### SEE ALSO

* [clickup tag](/clickup-cli/reference/clickup_tag/)	 - Manage space tags

