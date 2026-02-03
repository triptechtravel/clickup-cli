---
title: Home
layout: home
nav_order: 1
---

# clickup CLI

A command-line tool for working with ClickUp tasks, comments, and sprints -- designed for developers who live in the terminal and use GitHub.

`clickup` integrates with git to auto-detect task IDs from branch names and links GitHub pull requests, branches, and commits to ClickUp tasks.

---

## Features

- **Task management** -- view, list, create, and edit ClickUp tasks from the terminal.
- **Git integration** -- auto-detects task IDs from branch names (`CU-abc123` or `PROJ-42`).
- **GitHub linking** -- links PRs, branches, and commits to ClickUp tasks via comments.
- **Sprint dashboard** -- shows current sprint tasks grouped by status with assignees and priorities.
- **Inbox** -- surfaces recent @mentions across your workspace.
- **Fuzzy status matching** -- change task status with partial or fuzzy input.
- **JSON output** -- all list/view commands support `--json` and `--jq` for scripting.
- **Shell completions** -- bash, zsh, fish, and PowerShell.
- **Secure credentials** -- tokens stored in the system keyring.

## Quick install

```sh
go install github.com/triptechtravel/clickup-cli/cmd/clickup@latest
```

Or via Homebrew:

```sh
brew install triptechtravel/tap/clickup
```

See the [Installation](installation) page for all options.

## Quick start

```sh
# Authenticate with ClickUp
clickup auth login

# Select a default space
clickup space select

# View the task for your current git branch
clickup task view

# Set a task's status
clickup status set "in progress"

# Link your GitHub PR to the task
clickup link pr
```

See the [Getting started](getting-started) guide for a full walkthrough.

---

Created by [Isaac Rowntree](https://github.com/isaacrowntree) -- open source under [MIT](https://github.com/triptechtravel/clickup-cli/blob/main/LICENSE).
