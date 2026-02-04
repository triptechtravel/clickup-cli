# @triptechtravel/clickup-cli

A command-line tool for working with ClickUp tasks, comments, and sprints -- designed for developers who live in the terminal and use GitHub.

[![Release](https://img.shields.io/github/v/release/triptechtravel/clickup-cli)](https://github.com/triptechtravel/clickup-cli/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/triptechtravel/clickup-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/triptechtravel/clickup-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/triptechtravel/clickup-cli)](https://goreportcard.com/report/github.com/triptechtravel/clickup-cli)
[![Go Reference](https://pkg.go.dev/badge/github.com/triptechtravel/clickup-cli.svg)](https://pkg.go.dev/github.com/triptechtravel/clickup-cli)

`clickup` integrates with git to auto-detect task IDs from branch names and links GitHub pull requests, branches, and commits to ClickUp tasks.

## Features

- **Task management** -- view, list, create, and edit ClickUp tasks from the terminal
- **Custom fields** -- list, set, and clear custom field values on tasks (text, number, dropdown, labels, date, checkbox, URL, and more)
- **Dependencies & checklists** -- add/remove task dependencies and manage checklists with items
- **Git integration** -- auto-detects task IDs from branch names (`CU-abc123` or `PROJ-42`)
- **GitHub linking** -- links PRs, branches, and commits to ClickUp tasks via a managed description section rendered as rich text (or optional custom field)
- **Bidirectional sync** -- `link sync` pushes ClickUp task info into GitHub PR descriptions and vice versa
- **Sprint dashboard** -- shows current sprint tasks grouped by status with assignees and priorities
- **Inbox** -- surfaces recent @mentions across your workspace
- **Fuzzy status matching** -- change task status with partial or fuzzy input
- **Time tracking** -- log and view time entries on tasks
- **Full task properties** -- set tags, due dates, start dates, time estimates, story points, parent tasks, linked tasks, and task types from the CLI
- **AI-friendly** -- structured `--json` output and explicit `--task`/`--repo` flags make it easy for AI coding agents (Claude Code, Copilot, Cursor) to read ClickUp context and update tasks as part of a development workflow
- **GitHub Actions ready** -- automate status changes, PR linking, and task updates on PR events
- **JSON output** -- all list/view commands support `--json` and `--jq` for scripting
- **Shell completions** -- bash, zsh, fish, and PowerShell
- **Secure credentials** -- tokens stored in the system keyring with automatic expiration detection

## Installation

### Go

```sh
go install github.com/triptechtravel/clickup-cli/cmd/clickup@latest
```

### Homebrew

```sh
brew install triptechtravel/tap/clickup
```

### Binary releases

Download a prebuilt binary from the [releases page](https://github.com/triptechtravel/clickup-cli/releases) and add it to your `PATH`.

## Quick start

Authenticate with your ClickUp account:

```sh
clickup auth login
```

You will be prompted for a personal API token. Get one from **ClickUp > Settings > ClickUp API > API tokens**. The login flow also selects your default workspace.

Select a default space:

```sh
clickup space select
```

View the task associated with your current git branch:

```sh
clickup task view
```

Or specify a task ID directly:

```sh
clickup task view CU-abc123
```

Set a task's status with fuzzy matching:

```sh
clickup status set "in progress"
```

Link your current GitHub PR to the task:

```sh
clickup link pr
```

## Command reference

### Core commands

| Command | Description |
|---------|-------------|
| `clickup task view [task-id]` | View task details (auto-detects from branch) |
| `clickup task list --list-id ID` | List tasks with optional filters |
| `clickup task create --list-id ID` | Create a new task (interactive or flags) |
| `clickup task edit [task-id]` | Edit task fields (name, status, priority, dates, tags, points, custom fields, etc.) |
| `clickup task search <query>` | Search tasks with fuzzy matching |
| `clickup task recent` | Show your recently updated tasks with folder/list context |
| `clickup task activity [task-id]` | View task details and comment history |
| `clickup task time log [task-id]` | Log time to a task |
| `clickup task time list [task-id]` | View time entries for a task |
| `clickup task time delete <entry-id>` | Delete a time entry |
| `clickup task dependency add/remove` | Manage task dependencies (depends-on, blocks) |
| `clickup task checklist add/remove` | Manage task checklists and checklist items |
| `clickup comment add [task-id]` | Add a comment to a task |
| `clickup comment list [task-id]` | List comments on a task |
| `clickup comment edit` | Edit an existing comment |
| `clickup comment delete` | Delete a comment |
| `clickup status set STATUS [task-id]` | Change task status with fuzzy matching |
| `clickup status list` | List available statuses for a space |
| `clickup field list --list-id ID` | List available custom fields for a list |
| `clickup tag list` | List available tags for a space |

### Workflow commands

| Command | Description |
|---------|-------------|
| `clickup link pr [NUMBER]` | Link a GitHub PR to a ClickUp task |
| `clickup link sync [NUMBER]` | Sync ClickUp task info into a GitHub PR body (and link back) |
| `clickup link branch` | Link the current git branch to a task |
| `clickup link commit [SHA]` | Link a git commit to a task |
| `clickup sprint current` | Show tasks in the active sprint |
| `clickup sprint list` | List sprints in the sprint folder |
| `clickup inbox` | Show recent @mentions across your workspace |

### Setup commands

| Command | Description |
|---------|-------------|
| `clickup auth login` | Authenticate (token prompt, `--oauth`, or `--with-token` for CI) |
| `clickup auth logout` | Remove stored credentials |
| `clickup auth status` | Show current authentication state (includes user ID) |
| `clickup space list` | List spaces in your workspace |
| `clickup space select` | Interactively select a default space |
| `clickup member list` | List workspace members with IDs, usernames, emails, and roles |

### Utility commands

| Command | Description |
|---------|-------------|
| `clickup version` | Print version, commit, and build date |
| `clickup completion SHELL` | Generate shell completion scripts |

## Configuration

Configuration is stored in `~/.config/clickup/config.yml`. The file is created automatically during `clickup auth login`.

```yaml
workspace: "1234567"
space: "89012345"
sprint_folder: "67890123"
editor: "vim"
aliases:
  v: task view
  s: sprint current
directory_defaults:
  /home/user/projects/api:
    space: "11111111"
```

| Field | Description |
|-------|-------------|
| `workspace` | Default workspace (team) ID |
| `space` | Default space ID |
| `sprint_folder` | Folder ID containing sprint lists |
| `editor` | Editor for interactive descriptions |
| `prompt` | Controls interactive prompts (`"enabled"` by default) |
| `link_field` | Custom field name for storing GitHub links (optional; defaults to description section) |
| `aliases` | Custom command aliases |
| `directory_defaults` | Per-directory space and link_field overrides |

The config directory can be overridden with the `CLICKUP_CONFIG_DIR` environment variable.

## Git integration

The CLI auto-detects ClickUp task IDs from your current git branch name. Two patterns are recognized:

- **`CU-<id>`** -- default ClickUp task IDs (e.g., `CU-ae27de`, `CU-86d1u2bz4`)
- **`PREFIX-<number>`** -- custom task IDs (e.g., `PROJ-42`, `ENG-1234`)

Standard branch prefixes like `feature/`, `fix/`, `hotfix/`, `bugfix/`, `chore/`, and others are stripped before matching.

Name your branches with the task ID included:

```sh
git checkout -b feature/CU-ae27de-add-user-auth
git checkout -b fix/PROJ-42-login-bug
```

Then commands like `task view`, `task edit`, `status set`, `link pr`, `link branch`, and `link commit` will automatically target the correct task without requiring an explicit ID argument.

### GitHub CLI dependency

The `link pr` command requires the [GitHub CLI](https://cli.github.com/) (`gh`) to be installed and authenticated. It uses `gh pr view` to resolve pull request details.

## Shell completions

Generate completion scripts for your shell:

```sh
# Bash
source <(clickup completion bash)

# Zsh
source <(clickup completion zsh)
# Or install permanently:
clickup completion zsh > "${fpath[1]}/_clickup"

# Fish
clickup completion fish | source

# PowerShell
clickup completion powershell | Out-String | Invoke-Expression
```

## CI usage

For non-interactive environments, pipe a token via stdin:

```sh
echo "$CLICKUP_TOKEN" | clickup auth login --with-token
```

All list and view commands support `--json` for machine-readable output, and `--jq` for inline filtering:

```sh
clickup task view CU-abc123 --json
clickup sprint current --json --jq '.[].name'
```

## Using with AI coding agents

The CLI is designed to work well with AI agents like Claude Code, GitHub Copilot, and Cursor. An AI agent can read task context from ClickUp, make code changes, and update ClickUp -- all without leaving the terminal.

```sh
# AI agent discovers where work is happening
clickup task recent --json

# AI agent reads the task to understand requirements
clickup task view CU-abc123 --json

# Search within a specific folder/list discovered from recent tasks
clickup task search "migration" --folder "Engineering Sprint"

# After making changes, the agent updates ClickUp
clickup status set "code review" CU-abc123
clickup comment add CU-abc123 "Implemented the feature, PR is up for review"
clickup link sync --task CU-abc123
```

The `--json` flag on all commands outputs structured data that agents can parse. The `--task` and `--repo` flags on link commands allow targeting any task/repo without needing to be on the right branch.

When search returns no results, the CLI suggests `clickup task recent` to help discover active lists and folders. The `task recent` command shows each task's folder and list, so agents can quickly identify where to narrow their search.

## GitHub Actions

Automate ClickUp updates on PR events by adding workflow files to your repository. Copy these from the [`examples/`](https://github.com/triptechtravel/clickup-cli/tree/main/examples) directory or use them as a starting point.

### Sync task info on PR open

```yaml
# .github/workflows/clickup-sync.yml
name: ClickUp Sync
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install clickup CLI
        run: |
          curl -sL https://github.com/triptechtravel/clickup-cli/releases/latest/download/clickup_linux_amd64.tar.gz | tar xz
          sudo mv clickup /usr/local/bin/
      - name: Authenticate
        run: echo "${{ secrets.CLICKUP_TOKEN }}" | clickup auth login --with-token
      - name: Sync PR with ClickUp task
        run: clickup link sync ${{ github.event.pull_request.number }}
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Set status to "done" on merge

```yaml
# .github/workflows/clickup-done.yml
name: ClickUp Done
on:
  pull_request:
    types: [closed]

jobs:
  done:
    if: github.event.pull_request.merged == true
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install clickup CLI
        run: |
          curl -sL https://github.com/triptechtravel/clickup-cli/releases/latest/download/clickup_linux_amd64.tar.gz | tar xz
          sudo mv clickup /usr/local/bin/
      - name: Authenticate
        run: echo "${{ secrets.CLICKUP_TOKEN }}" | clickup auth login --with-token
      - name: Set task to done
        run: clickup status set "done"
```

### Comment CI results on task

```yaml
# .github/workflows/clickup-ci.yml
name: ClickUp CI Status
on:
  check_suite:
    types: [completed]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install clickup CLI
        run: |
          curl -sL https://github.com/triptechtravel/clickup-cli/releases/latest/download/clickup_linux_amd64.tar.gz | tar xz
          sudo mv clickup /usr/local/bin/
      - name: Authenticate
        run: echo "${{ secrets.CLICKUP_TOKEN }}" | clickup auth login --with-token
      - name: Post CI result
        run: |
          STATUS="${{ github.event.check_suite.conclusion }}"
          clickup comment add "" "CI ${STATUS}: ${{ github.event.check_suite.head_branch }} (${{ github.sha }})"
```

## Documentation

Full documentation is available at [triptechtravel.github.io/clickup-cli](https://triptechtravel.github.io/clickup-cli/).

## Author

Created by [Isaac Rowntree](https://github.com/isaacrowntree).

## License

[MIT](LICENSE)
