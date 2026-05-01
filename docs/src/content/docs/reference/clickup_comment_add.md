---
title: "clickup comment add"
description: "Auto-generated reference for clickup comment add"
---

Add a comment to a task

### Synopsis

Add a comment to a ClickUp task.

If TASK is not provided, the task ID is auto-detected from the current git branch.
If BODY is not provided (or --editor is used), your editor opens for composing the comment.

The body is parsed as markdown. Headers (##), bold (**x**), italic (*x*),
inline code, fenced code blocks, ordered/bullet lists, blockquotes, and links
are all rendered as rich formatting in ClickUp.

Use @username in the body to @mention workspace members. Mentions resolve
against your workspace member list (see 'clickup member list') with
case-insensitive matching, and additionally accept the first-name token or
email local-part when unambiguous within the workspace.

```
clickup comment add [TASK] [BODY] [flags]
```

### Examples

```
  # Add a comment to the task detected from the current branch
  clickup comment add "" "Fixed the login bug"

  # Add a comment to a specific task
  clickup comment add abc123 "Deployed to staging"

  # Mention a teammate (triggers a real ClickUp notification)
  clickup comment add abc123 "Hey @alice can you review this?"

  # Mention multiple people
  clickup comment add abc123 "@alice @bob this is ready for QA"

  # Open your editor to compose the comment
  clickup comment add --editor
```

### Options

```
  -e, --editor            Open editor to compose comment body
  -h, --help              help for add
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
  -r, --raw               Output raw strings instead of JSON-encoded (use with --jq)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup comment](/clickup-cli/reference/clickup_comment/)	 - Manage comments on ClickUp tasks

