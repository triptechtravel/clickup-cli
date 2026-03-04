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
- **Inbox** -- surfaces recent @mentions in comments and task descriptions across your workspace
- **Fuzzy status matching** -- change task status with partial or fuzzy input
- **Time tracking** -- log and view time entries on tasks; workspace-wide timesheet queries with date ranges
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

## Command overview

| Area | Key commands |
|------|-------------|
| **Tasks** | `task view`, `task list`, `task create`, `task edit`, `task search`, `task recent`, `task delete` |
| **Time tracking** | `task time log`, `task time list`, `task time delete` |
| **Comments** | `comment add`, `comment list`, `comment edit`, `comment delete` |
| **Status** | `status set`, `status list`, `status add` |
| **Git & GitHub** | `link pr`, `link sync`, `link branch`, `link commit` |
| **Sprints** | `sprint current`, `sprint list` |
| **Workspace** | `inbox`, `member list`, `space list`, `space select`, `tag list`, `field list` |
| **Auth** | `auth login`, `auth logout`, `auth status` |

All list/view commands support `--json` and `--jq` for scripting. See the [full command reference](https://triptechtravel.github.io/clickup-cli/commands/) for flags, examples, and detailed usage.

## Git integration

The CLI auto-detects ClickUp task IDs from branch names (`CU-abc123` or `PROJ-42`). Name your branches with the task ID:

```sh
git checkout -b feature/CU-ae27de-add-user-auth
```

Then `task view`, `task edit`, `status set`, `link pr`, and other commands automatically target the correct task. See the [git integration guide](https://triptechtravel.github.io/clickup-cli/git-integration/) for details.

## Documentation

Full documentation -- including [configuration](https://triptechtravel.github.io/clickup-cli/configuration/), [CI usage](https://triptechtravel.github.io/clickup-cli/ci-usage/), [GitHub Actions](https://triptechtravel.github.io/clickup-cli/github-actions/), [AI agent integration](https://triptechtravel.github.io/clickup-cli/ai-agents/), and [auto-generated command reference](https://triptechtravel.github.io/clickup-cli/reference/clickup/) -- is available at **[triptechtravel.github.io/clickup-cli](https://triptechtravel.github.io/clickup-cli/)**.

## Author

Created by [Isaac Rowntree](https://github.com/isaacrowntree).

## License

[MIT](LICENSE)
