# clickup-cli (nick-preda fork)

A CLI for managing ClickUp from the terminal. Forked from [triptechtravel/clickup-cli](https://github.com/triptechtravel/clickup-cli) with extra features for daily use with AI agents (Claude Code) and automation.

## What this fork adds

| Feature | Command | Why |
|---------|---------|-----|
| **List all lists in a space** | `clickup list ls` | Find list IDs without digging through the UI |
| **Send messages to Chat channels** | `clickup chat send <channel-id> "msg"` | Post reports, alerts, and notifications to Chat |
| **Create tasks by list name** | `clickup task create --list-name "Issues"` | No need to look up numeric list IDs |
| **Search with assignee filter** | `clickup task search "term" --assignee me` | Filter search results by assignee |
| **Faster search** | Server-side search + parallel space traversal | Upstream only did client-side filtering |

## Install

```sh
# From source (recommended for this fork)
git clone https://github.com/nick-preda/clickup-cli.git
cd clickup-cli
make install

# Or directly with Go
go install github.com/nick-preda/clickup-cli/cmd/clickup@latest
```

## Quick start

```sh
clickup auth login         # authenticate with your API token
clickup space select       # choose a default space
clickup list ls            # see all lists and their IDs
clickup task create --list-name "Issues" --name "Fix the bug" --priority 2
clickup task search "bug"  # find tasks
clickup chat send khpgh-10335 "Deploy done"  # post to a Chat channel
```

## Daily workflow

```sh
# Find where to create a task
clickup list ls
# 900601764492  Issues      (no folder)
# 900401327544  Nanea       (no folder)
# 901510841332  Task        Gitlab

# Create a task by name (no list-id needed)
clickup task create --list-name "Issues" \
  --name "[Bug] Fix login timeout" --priority 2

# Search your tasks
clickup task search "login" --assignee me

# Add a comment with @mentions
clickup comment add 86abc123 "@Michela this is ready for review"

# Send a report to a Chat channel
clickup chat send khpgh-10335 "Daily report: all systems green"

# View task from current git branch (auto-detected)
clickup task view
```

## All commands

| Area | Commands |
|------|----------|
| **Tasks** | `task view`, `task create`, `task edit`, `task search`, `task list`, `task recent`, `task delete` |
| **Lists** | `list ls` |
| **Chat** | `chat send` |
| **Comments** | `comment add`, `comment list`, `comment edit`, `comment delete` |
| **Status** | `status set`, `status list`, `status add` |
| **Git** | `link pr`, `link sync`, `link branch`, `link commit` |
| **Sprints** | `sprint current`, `sprint list` |
| **Workspace** | `space list`, `space select`, `member list`, `inbox`, `tag list`, `field list` |

## Using with AI agents

This CLI is designed to work well with Claude Code and other AI agents:

```sh
# JSON output for programmatic use
clickup list ls --json
clickup task search "deploy" --json

# AI agent can create tasks without knowing list IDs
clickup task create --list-name "Issues" --name "task name"

# AI agent can post to Chat channels
clickup chat send <channel-id> "automated report here"
```

## Configuration

Config is stored in `~/.config/clickup/config.yml`:

```yaml
workspace: "20503057"        # your team/workspace ID
space: "90060297766"         # default space for list ls, task create --list-name
```

Set per-directory defaults with `directory_defaults` in the config file.

## Upstream docs

For features inherited from upstream, see the [original documentation](https://triptechtravel.github.io/clickup-cli/).

## License

[MIT](LICENSE)
