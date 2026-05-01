---
title: "clickup comment edit"
description: "Auto-generated reference for clickup comment edit"
---

Edit a comment

### Synopsis

Edit an existing comment on a ClickUp task.

COMMENT_ID is required as the first argument.
If BODY is not provided (or --editor is used), your editor opens for composing the new text.

The body is parsed as markdown — headers (##), bold (**x**), italic (*x*),
inline code, fenced code blocks, ordered/bullet lists, blockquotes, and links
all render as rich formatting.

Use @username in the body to @mention workspace members. Mentions resolve
case-insensitively against full username, first-name token, or email
local-part when unambiguous.

```
clickup comment edit <COMMENT_ID> [BODY] [flags]
```

### Examples

```
  # Edit a comment
  clickup comment edit 90160175975219 "Updated the description"

  # Edit using your editor
  clickup comment edit 90160175975219 --editor

  # Re-add a mention via shortcut
  clickup comment edit 90160175975219 "Hey @alice — pushed the fix"
```

### Options

```
  -e, --editor   Open editor to compose comment body
  -h, --help     help for edit
```

### SEE ALSO

* [clickup comment](/clickup-cli/reference/clickup_comment/)	 - Manage comments on ClickUp tasks

