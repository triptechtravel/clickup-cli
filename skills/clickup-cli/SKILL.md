---
name: clickup-cli
description: ClickUp CLI for managing tasks, sprints, comments, and statuses. Use when the user needs to interact with ClickUp — creating/editing tasks, checking sprint status, adding comments, linking PRs, or searching tasks. Prefer this CLI over raw API calls.
---

# ClickUp CLI (`clickup`)

Use the `clickup` CLI instead of raw ClickUp API calls. It handles authentication, git integration, fuzzy status matching, and custom fields automatically.

## When to Use

- User asks to create, edit, view, or search ClickUp tasks
- User wants to check sprint status or recent tasks
- User needs to add comments, link PRs/branches, or manage task statuses
- User mentions ClickUp task IDs (e.g., `CU-abc123`, `86abc123`)
- User asks about their ClickUp inbox or mentions

## Authentication

```bash
clickup auth login    # Authenticate with API token
clickup auth status   # Check auth status
```

Configuration is stored in `~/.config/clickup/config.yml`. Supports per-directory defaults for space, team, and folder.

## Task Management

### View & Search

```bash
# View a task (auto-detects from git branch if no ID given)
clickup task view
clickup task view CU-abc123

# Search tasks by name (supports fuzzy matching)
clickup task search "login bug"
clickup task search "login bug" --exact    # Exact matches only

# Recent tasks (excludes archived folders)
clickup task recent
clickup task recent --sprint               # Only current sprint tasks

# List tasks in a specific list
clickup task list --list-id 12345

# Task activity/comment history
clickup task activity CU-abc123
```

### Create & Edit

**IMPORTANT: When creating a task, fill in ALL applicable fields.** Do not create bare tasks with just a name. Ask the user for any information you don't already have. A well-created task should include as many of these as possible:

- `--name` — task name (required)
- `--description` or `--markdown-description` — clear description of the work
- `--status` — initial status (e.g., "open", "in progress")
- `--priority` — 1=Urgent, 2=High, 3=Normal, 4=Low
- `--assignee` — user ID(s) for who should work on it
- `--tags` — relevant tags (repeatable)
- `--due-date` — deadline (YYYY-MM-DD)
- `--start-date` — when work should begin (YYYY-MM-DD)
- `--time-estimate` — estimated effort (e.g., "2h", "4h", "1d")
- `--points` — sprint/story points
- `--parent` — parent task ID if this is a subtask
- `--links-to` — related task ID
- `--type` — 0=task, 1=milestone
- `--field "Name=value"` — custom fields (repeatable)

After creating a task, consider adding checklists for acceptance criteria or subtasks:

```bash
clickup task checklist add <task-id> "Acceptance Criteria"
clickup task checklist item add <checklist-id> "Unit tests pass"
clickup task checklist item add <checklist-id> "Code reviewed"
```

Example of a well-populated task creation:

```bash
clickup task create --list-id 12345 \
  --name "Fix login timeout on slow connections" \
  --markdown-description "Users on slow 3G connections get a timeout error..." \
  --status "open" \
  --priority 2 \
  --assignee 12345678 \
  --tags "bug" --tags "auth" \
  --due-date 2025-03-01 \
  --start-date 2025-02-20 \
  --time-estimate 4h \
  --points 3
```

```bash
# Edit a task (auto-detects from git branch)
clickup task edit --status "in progress" --priority 2
clickup task edit CU-abc123 --field "Environment=production"
clickup task edit --due-date 2025-03-01 --time-estimate 4h

# Custom fields
clickup task edit CU-abc123 --field "Environment=production"
clickup task edit CU-abc123 --clear-field "Environment"
clickup field list --list-id 12345    # Discover available fields
```

### Status Management

```bash
# Set status (supports fuzzy matching: "review" matches "code review")
clickup status set "in progress"
clickup status set "in progress" CU-abc123

# List available statuses
clickup status list
clickup status list --space 12345
```

Status values are fuzzy-matched: exact match > contains match > fuzzy match. If ambiguous, the CLI picks the most specific match and prints a warning.

## Sprints

```bash
# Show current sprint tasks
clickup sprint current

# List all sprints in a folder
clickup sprint list
```

## Comments

```bash
# Add a comment (supports @mentions — resolves usernames to ClickUp user tags)
clickup comment add CU-abc123 "Looks good, @alice please review"

# List comments
clickup comment list CU-abc123

# Edit/delete comments
clickup comment edit <comment-id> "Updated text"
clickup comment delete <comment-id>
```

## Git & GitHub Integration

The CLI auto-detects task IDs from git branch names. Branch naming convention: `feature/CU-abc123-description` or `CU-abc123/description`.

```bash
# Link a GitHub PR to a ClickUp task
clickup link pr
clickup link pr --task CU-abc123

# Link a specific PR number to a task (useful after merging)
clickup link pr 42 --task CU-abc123

# Link current branch
clickup link branch

# Link a commit
clickup link commit

# Sync ClickUp task info to GitHub PR description
clickup link sync
clickup link sync --task CU-abc123
clickup link sync 42 --repo owner/repo --task CU-abc123
```

Links are stored in the task's markdown description as rich-text with clickable URLs.

**Note:** When `--task` is specified but no PR number, the CLI first tries the current branch's PR, then searches for PRs matching the task ID in their branch name. This works even after merging when the feature branch is deleted.

## Time Tracking

```bash
# Log time to a task (auto-detects from git branch)
clickup task time log --duration 2h
clickup task time log 86abc123 --duration 1h30m --description "Implemented auth flow"

# Log time for a specific date
clickup task time log --duration 45m --date 2025-01-15

# Log billable time
clickup task time log --duration 3h --billable

# List time entries for a task
clickup task time list
clickup task time list 86abc123
clickup task time list 86abc123 --json
```

## Inbox

```bash
# Show @mentions from the last 7 days (scans comments and task descriptions)
clickup inbox
clickup inbox --days 30
clickup inbox --limit 500    # Scan more tasks in busy workspaces
```

## Workspace

```bash
# List workspace members
clickup member list

# List/select spaces
clickup space list
clickup space select    # Set default space
```

## Common Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq <expr>` | Filter JSON with jq expression |
| `--template <tmpl>` | Format with Go template |

## Key Behaviors

- **Auto-detection**: Most commands auto-detect task ID from the current git branch
- **Fuzzy status matching**: Status values are fuzzy-matched against available statuses
- **Status validation**: `task create` and `task edit` validate statuses against the space's configured statuses
- **Archive filtering**: `task recent` automatically excludes tasks from archived folders
- **Custom IDs**: Supports both native IDs and custom IDs (e.g., `CU-abc123`)
- **@mentions**: Comment add resolves `@username` to real ClickUp user tags
