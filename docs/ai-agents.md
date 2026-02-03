---
title: AI agents
layout: default
nav_order: 5.8
---

# Using with AI agents

The CLI is designed to work with AI coding agents like Claude Code, GitHub Copilot, and Cursor. Every command supports structured output and explicit targeting, so an agent can read ClickUp context, write code, and update tasks without leaving the terminal.

## Why this matters

AI agents work best when they have full context about what they're building. Instead of copy-pasting task descriptions into a chat window, the agent can pull requirements directly from ClickUp, then update the task when the work is done.

## Typical AI workflow

```sh
# 1. Agent reads the task to understand requirements
clickup task view CU-abc123 --json

# 2. Agent reads comments for additional context
clickup comment list CU-abc123

# 3. Agent writes code...

# 4. Agent updates the task status
clickup status set "code review" CU-abc123

# 5. Agent adds a comment summarizing what was done
clickup comment add CU-abc123 "Implemented auth flow with JWT tokens. PR #42 is up."

# 6. Agent syncs the ClickUp task info into the PR description
clickup link sync 42 --task CU-abc123
```

## Key features for AI agents

### Structured JSON output

All list and view commands support `--json` for machine-readable output:

```sh
# Get task details as JSON
clickup task view CU-abc123 --json

# Get current sprint tasks as JSON
clickup sprint current --json

# Filter with jq expressions
clickup sprint current --json --jq '[.[] | select(.status == "to do")]'
```

### Explicit targeting with `--task` and `--repo`

Link commands accept `--task` and `--repo` flags so agents don't need to be on a specific branch:

```sh
# Link a PR from any repo to any task
clickup link pr 42 --repo owner/repo --task CU-abc123

# Sync a PR with a task regardless of current directory
clickup link sync 42 --repo owner/repo --task CU-abc123
```

### Fuzzy status matching

Agents don't need to know exact status names. The CLI fuzzy-matches:

```sh
clickup status set "prog" CU-abc123    # matches "in progress"
clickup status set "review" CU-abc123  # matches "code review"
clickup status set "done" CU-abc123    # matches "done"
```

### Task ID auto-detection

When the agent is working in a repo with a properly named branch, no task ID is needed:

```sh
# On branch feature/CU-abc123-add-auth
clickup task view          # auto-detects CU-abc123
clickup status set "done"  # auto-detects CU-abc123
clickup link pr            # auto-detects CU-abc123
```

## Example: Claude Code integration

When using Claude Code, the agent can be instructed to use the CLI as part of its workflow:

```
Task: Implement the feature described in ClickUp task CU-abc123

1. Run `clickup task view CU-abc123` to read the requirements
2. Implement the feature
3. Run `clickup status set "code review" CU-abc123`
4. Run `clickup comment add CU-abc123 "summary of changes"`
5. Run `clickup link sync --task CU-abc123` to update the PR
```

The CLI handles authentication via the system keyring, so no tokens need to be passed in prompts.

## Example: CI/CD with AI-generated PRs

When AI agents create PRs automatically, combine the CLI with GitHub Actions to keep ClickUp in sync:

```yaml
# After AI agent pushes a branch and creates a PR
- name: Sync with ClickUp
  run: |
    clickup link sync ${{ github.event.pull_request.number }}
    clickup status set "code review"
    clickup comment add "" "AI-generated PR #${{ github.event.pull_request.number }} created"
```

## Tips

- Use `--json` output when you need the agent to parse task data programmatically
- Use `clickup comment list` to give the agent context from team discussions
- Use `clickup sprint current --json` to help the agent understand project priorities
- The `link sync` command is idempotent -- safe to run multiple times without duplicating data
