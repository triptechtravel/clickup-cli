---
title: Configuration
description: Config file reference, per-directory defaults, aliases, and environment variables.
---

# Configuration

The CLI stores configuration in a YAML file at `~/.config/clickup/config.yml`. This file is created automatically during `clickup auth login`.

## Config file reference

```yaml
workspace: "1234567"
space: "89012345"
sprint_folder: "67890123"
editor: "vim"
prompt: "enabled"
link_field: "GitHub Links"
aliases:
  v: task view
  s: sprint current
directory_defaults:
  /home/user/projects/api:
    space: "11111111"
    link_field: "PR Link"
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `workspace` | string | Default workspace (team) ID. Set during `auth login`. |
| `space` | string | Default space ID. Set via `space select`. |
| `sprint_folder` | string | Folder ID containing sprint lists. Auto-detected or set via `--folder`. |
| `editor` | string | Editor command for interactive descriptions and comments. Falls back to `$EDITOR`. |
| `prompt` | string | Controls interactive prompts. Set to `"enabled"` by default. |
| `link_field` | string | Custom field name for storing GitHub links from `link` commands. When not set, links are stored in a managed section of the task description. See [GitHub link storage](#github-link-storage) below. |
| `aliases` | map | Custom command aliases. Keys are alias names, values are the full command string. |
| `directory_defaults` | map | Per-directory configuration overrides (see below). |

## Per-directory defaults

You can configure different spaces for different project directories. When you run a command from a directory that has an entry in `directory_defaults`, the CLI uses that directory's space instead of the global default.

```yaml
directory_defaults:
  /home/user/projects/api:
    space: "11111111"
  /home/user/projects/frontend:
    space: "22222222"
```

Each directory entry supports the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `space` | string | Space ID to use when running commands from this directory. |
| `link_field` | string | Custom field name for GitHub links, overriding the global `link_field` for this directory. |

The CLI checks the current working directory against the `directory_defaults` map. If a match is found, the directory-specific values override the global settings.

## Custom aliases

Define aliases to create shortcuts for frequently used commands:

```yaml
aliases:
  v: task view
  s: sprint current
  tl: task list --list-id 12345
```

## GitHub link storage

The `link pr`, `link branch`, `link commit`, and `link sync` commands store GitHub links on ClickUp tasks. There are two storage approaches:

### Description section (default)

When `link_field` is **not configured**, links are stored in a managed block within the task description:

```
--- GitHub Links (clickup-cli) ---
PR: owner/repo#42 - Fix login bug (https://github.com/owner/repo/pull/42)
Branch: feat/fix-login in owner/repo
--- /GitHub Links ---
```

Each link entry is deduplicated by a prefix key (e.g., `PR: owner/repo#42`), so re-running the same command updates the existing line rather than adding a duplicate. Multiple PRs from different repos coexist as separate entries.

### Custom field (optional)

When `link_field` is configured, links are stored in the named custom field instead. This works with two ClickUp field types:

- **URL/Website fields**: The field is set to the latest link URL. Best for tasks with a single PR.
- **Text fields**: Link entries are appended as separate lines with deduplication. Best for cross-cutting tasks with multiple PRs across repos.

To set up custom field linking:

1. Create a custom field in ClickUp (e.g., a "Website" field named "PR Link")
2. Add the field name to your config:

```yaml
link_field: "PR Link"
```

Use `clickup field list --list-id ID` to see available custom fields and their types.

## Environment variables

| Variable | Description |
|----------|-------------|
| `CLICKUP_CONFIG_DIR` | Override the config directory path. Default: `~/.config/clickup`. |

When `CLICKUP_CONFIG_DIR` is set, the CLI reads and writes `config.yml` from that directory instead of the default location.

## Config file location

The config file path is determined as follows:

1. If `CLICKUP_CONFIG_DIR` is set, use `$CLICKUP_CONFIG_DIR/config.yml`.
2. Otherwise, use `~/.config/clickup/config.yml`.

The directory is created automatically if it does not exist.
