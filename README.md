# clickup

A command-line tool for working with ClickUp tasks, comments, and sprints -- designed for developers who live in the terminal and use GitHub.

`clickup` integrates with git to auto-detect task IDs from branch names and links GitHub pull requests, branches, and commits to ClickUp tasks.

## Features

- **Task management** -- view, list, create, and edit ClickUp tasks from the terminal
- **Git integration** -- auto-detects task IDs from branch names (`CU-abc123` or `PROJ-42`)
- **GitHub linking** -- links PRs, branches, and commits to ClickUp tasks via comments
- **Sprint dashboard** -- shows current sprint tasks grouped by status with assignees and priorities
- **Inbox** -- surfaces recent @mentions across your workspace
- **Fuzzy status matching** -- change task status with partial or fuzzy input
- **JSON output** -- all list/view commands support `--json` and `--jq` for scripting
- **Shell completions** -- bash, zsh, fish, and PowerShell
- **Secure credentials** -- tokens stored in the system keyring

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
| `clickup task edit [task-id]` | Edit task fields (name, status, priority, etc.) |
| `clickup comment add [task-id]` | Add a comment to a task |
| `clickup comment list [task-id]` | List comments on a task |
| `clickup comment edit` | Edit an existing comment |
| `clickup status set STATUS [task-id]` | Change task status with fuzzy matching |
| `clickup status list` | List available statuses for a space |

### Workflow commands

| Command | Description |
|---------|-------------|
| `clickup link pr [NUMBER]` | Link a GitHub PR to a ClickUp task |
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
| `clickup auth status` | Show current authentication state |
| `clickup space list` | List spaces in your workspace |
| `clickup space select` | Interactively select a default space |

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
| `aliases` | Custom command aliases |
| `directory_defaults` | Per-directory space overrides |

The config directory can be overridden with the `CLICKUP_CONFIG_DIR` environment variable.

## Git integration

The CLI auto-detects ClickUp task IDs from your current git branch name. Two patterns are recognized:

- **`CU-<hex>`** -- default ClickUp task IDs (e.g., `CU-ae27de`)
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

## Documentation

Full documentation is available at [triptechtravel.github.io/clickup-cli](https://triptechtravel.github.io/clickup-cli/).

## Author

Created by [Isaac Rowntree](https://github.com/isaacrowntree).

## License

[MIT](LICENSE)
