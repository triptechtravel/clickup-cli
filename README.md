# @triptechtravel/clickup-cli

A command-line tool for working with ClickUp tasks, comments, and sprints -- designed for developers who live in the terminal and use GitHub.

[![Release](https://img.shields.io/github/v/release/triptechtravel/clickup-cli)](https://github.com/triptechtravel/clickup-cli/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/triptechtravel/clickup-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/triptechtravel/clickup-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/triptechtravel/clickup-cli)](https://goreportcard.com/report/github.com/triptechtravel/clickup-cli)
[![Go Reference](https://pkg.go.dev/badge/github.com/triptechtravel/clickup-cli.svg)](https://pkg.go.dev/github.com/triptechtravel/clickup-cli)

## Install

```sh
# Homebrew
brew install triptechtravel/tap/clickup

# Go
go install github.com/triptechtravel/clickup-cli/cmd/clickup@latest

# Or download a binary from the releases page
```

## Quick start

```sh
clickup auth login        # authenticate with your API token
clickup space select       # choose a default space
clickup task view          # view the task from your current git branch
clickup status set "done"  # fuzzy-matched status update
clickup link pr            # link the current GitHub PR to the task
```

See the [getting started guide](https://triptechtravel.github.io/clickup-cli/getting-started/) for a full walkthrough.

## What it does

- **Task management** -- view, create, edit, search, and bulk-edit tasks with custom fields, tags, points, and time estimates
- **Docs** -- list, view, create ClickUp Docs and manage their pages (list, view, create, edit) via the v3 API
- **Git integration** -- auto-detects task IDs from branch names and links PRs, branches, and commits to ClickUp
- **Sprint dashboard** -- `sprint current` shows tasks grouped by status; `task create --current` creates tasks in the active sprint
- **Time tracking** -- log time, view per-task entries, or query workspace-wide timesheets by date range
- **Comments & inbox** -- add comments with @mentions, view your recent mentions across the workspace
- **Fuzzy status matching** -- set statuses with partial input (`"review"` matches `"code review"`)
- **AI-friendly** -- `--json` output and explicit flags make it easy for AI agents to read and update tasks
- **CI/CD ready** -- `--with-token`, exit codes, and JSON output for automation; includes GitHub Actions examples

## Commands

Full command list with flags and examples: **[Command reference](https://triptechtravel.github.io/clickup-cli/commands/)**

| Area | Key commands |
|------|-------------|
| **Tasks** | `task view`, `task create`, `task edit`, `task search`, `task recent` |
| **Docs** | `doc list`, `doc view`, `doc create`, `doc page list`, `doc page view`, `doc page create`, `doc page edit` |
| **Time** | `task time log`, `task time list` |
| **Status** | `status set`, `status list`, `status add` |
| **Git** | `link pr`, `link sync`, `link branch`, `link commit` |
| **Sprints** | `sprint current`, `sprint list` |
| **Comments** | `comment add`, `comment list` |
| **Attachments** | `attachment list`, `attachment add` |
| **Workspace** | `inbox`, `member list`, `space select`, `tag list`, `field list` |

## Documentation

**[triptechtravel.github.io/clickup-cli](https://triptechtravel.github.io/clickup-cli/)**

- [Installation](https://triptechtravel.github.io/clickup-cli/installation/) -- Homebrew, Go, binaries, shell completions
- [Getting started](https://triptechtravel.github.io/clickup-cli/getting-started/) -- first-time setup walkthrough
- [Configuration](https://triptechtravel.github.io/clickup-cli/configuration/) -- config file, per-directory defaults, aliases
- [Git integration](https://triptechtravel.github.io/clickup-cli/git-integration/) -- branch naming, GitHub linking strategy
- [CI usage](https://triptechtravel.github.io/clickup-cli/ci-usage/) -- non-interactive auth, JSON output, scripting
- [GitHub Actions](https://triptechtravel.github.io/clickup-cli/github-actions/) -- ready-to-use workflow templates
- [AI agents](https://triptechtravel.github.io/clickup-cli/ai-agents/) -- integration with Claude Code, Copilot, Cursor
- [Command reference](https://triptechtravel.github.io/clickup-cli/reference/clickup/) -- auto-generated flag and usage docs

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, project structure, and guidelines.

## Author

Created by [Isaac Rowntree](https://github.com/isaacrowntree).

## License

[MIT](LICENSE)
