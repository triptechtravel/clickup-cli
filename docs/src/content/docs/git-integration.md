---
title: Git integration
description: How the CLI auto-detects ClickUp task IDs from git branch names.
---

# Git integration

The CLI auto-detects ClickUp task IDs from your current git branch name. This means most commands work without specifying a task ID explicitly, as long as your branch name includes one.

## Supported patterns

Two task ID patterns are recognized:

### Default ClickUp IDs

Pattern: `CU-<hex>`

Matches the default alphanumeric hex IDs that ClickUp assigns to tasks. The match is case-insensitive.

Examples:
- `CU-ae27de`
- `CU-1a2b3c`
- `cu-FF00AA`

### Custom task IDs

Pattern: `PREFIX-<number>`

Matches custom task ID prefixes configured in your ClickUp workspace. The prefix must start with an uppercase letter followed by uppercase letters or digits, then a hyphen, then one or more digits.

Examples:
- `PROJ-42`
- `ENG-1234`
- `API-7`

## Branch naming conventions

Include the task ID anywhere in your branch name. Standard branch prefixes are stripped before pattern matching, so you can use any common branching convention:

```sh
git checkout -b feature/CU-ae27de-add-user-auth
git checkout -b fix/PROJ-42-login-bug
git checkout -b hotfix/ENG-100-critical-patch
git checkout -b chore/CU-1a2b3c-update-deps
```

### Recognized branch prefixes

The following prefixes are stripped automatically before the CLI searches for a task ID:

- `feature/`
- `fix/`
- `hotfix/`
- `bugfix/`
- `release/`
- `chore/`
- `docs/`
- `refactor/`
- `test/`
- `ci/`

### Excluded prefixes

Certain uppercase words are excluded from custom ID matching to avoid false positives. These correspond to common branch prefix conventions written in uppercase:

`FEATURE`, `BUGFIX`, `RELEASE`, `HOTFIX`, `FIX`, `CHORE`, `DOCS`, `REFACTOR`, `TEST`

For example, a branch named `FEATURE-123` will not match as a custom task ID.

## Detection priority

When scanning a branch name, the CLI applies patterns in this order:

1. **CU-hex** -- checked first. If a `CU-<hex>` pattern is found, it is used immediately.
2. **PREFIX-number** -- checked second. If a custom `PREFIX-<number>` pattern is found (and the prefix is not in the excluded list), it is used.

If neither pattern matches, the command reports that no task ID was found and suggests a branch naming format.

## Commands that use auto-detection

The following commands auto-detect the task ID from the branch when no explicit ID is provided:

- `task view`
- `task edit`
- `task activity`
- `task time log`
- `task time list`
- `comment add`
- `status set`
- `link pr`
- `link sync`
- `link branch`
- `link commit`

## GitHub linking strategy

The `link` commands connect GitHub artifacts to ClickUp tasks idempotently. Links are written via ClickUp's `markdown_description` API field, so they render as rich text with clickable links, bold formatting, and code blocks directly in the ClickUp UI.

By default, links are stored in a managed section of the task description. Optionally, you can configure a `link_field` to store links in a custom field instead (see [Configuration](/clickup-cli/configuration/#github-link-storage)).

Each link type produces a different entry in the description:

### `link pr`

```markdown
[owner/repo#42 — Fix authentication flow](https://github.com/owner/repo/pull/42)
```

Renders as a clickable link in ClickUp. Requires the [GitHub CLI](https://cli.github.com/) (`gh`) to be installed and authenticated.

### `link branch`

```markdown
Branch: `feature/CU-ae27de-add-auth` in owner/repo
```

Branch names are rendered in code formatting.

### `link commit`

```markdown
[`a1b2c3d` — Implement login validation](https://github.com/owner/repo/commit/fullsha)
```

Renders as a clickable link with the short SHA in code formatting.

Re-running any link command updates the existing entry rather than creating a duplicate. Multiple PRs from different repos coexist as separate entries, which is useful for cross-cutting tasks that span multiple repositories.

## Tips

- Always include the task ID near the beginning of the branch name for reliable detection.
- Use `CU-` prefix IDs when working with default ClickUp task IDs.
- Use custom prefix IDs (like `PROJ-42`) when your workspace has custom task ID prefixes enabled.
- The task ID can appear anywhere after the branch prefix -- `feature/CU-abc123-description` and `feature/some-description-CU-abc123` both work.
