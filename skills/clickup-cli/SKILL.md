---
name: clickup-cli
description: ClickUp CLI for managing tasks, sprints, comments, statuses, and Docs. Use when the user needs to interact with ClickUp — creating/editing tasks, checking sprint status, adding comments, linking PRs, managing Docs and pages, or searching tasks. Prefer this CLI over raw API calls.
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
# Also detects tasks via PR URL in descriptions when branch has no task ID
# Subtask lines show due/start dates inline for at-a-glance visibility
clickup task view
clickup task view CU-abc123

# View with JSON — useful for extracting subtask IDs for bulk operations
clickup task view 86abc123 --json

# Search tasks by name and description (supports fuzzy matching)
# Uses progressive drill-down: sprint → your tasks → space → workspace
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

**Navigating task hierarchies:** Use `--json` to drill into subtask trees. The JSON output includes a `subtasks` array with each subtask's `id`, `name`, `status`, `due_date`, and `start_date`. To operate on subtasks in bulk, view the parent with `--json`, extract the subtask IDs, then pass them to `task edit`.

### Task Naming Conventions

**IMPORTANT: Always use descriptive, well-structured task names.** Task names should be scannable on a sprint board without needing to click into the task. Follow this convention:

**Format:** `[Work Type] Context — Action (Platform/Scope)`

| Component | Purpose | Examples |
|-----------|---------|----------|
| `[Work Type]` | Categorises the work at a glance | `[Packdown]`, `[Bug]`, `[Feature]`, `[Refactor]`, `[Spike]`, `[Hotfix]` |
| `Context` | Campaign, project, or feature name | `NT x THL`, `Auth v2`, `Booking Flow` |
| `Action` | What is being done — use imperative verbs | `Remove landing page`, `Add SSO support`, `Fix timeout error` |
| `(Platform/Scope)` | Which app, site, or system is affected | `(CamperMate)`, `(Britz + Apollo)`, `(API)`, `(CamperMate + CM.com)` |

**Examples:**
- `[Packdown] NT x THL — Remove CamperMate campaign landing page`
- `[Packdown] NT x THL — Remove app menu links (CamperMate + CM.com)`
- `[Bug] Booking Flow — Fix timeout on slow connections (API)`
- `[Feature] Auth v2 — Add SSO support (CamperMate)`
- `[Spike] Search — Evaluate Algolia vs Typesense`

**When to simplify:** For small, self-evident tasks (e.g. subtasks or checklists), a short descriptive name is fine. The full convention is most valuable for sprint-level tasks that need to be scannable across teams.

**Template tasks:** Use `{Campaign}` or `{Project}` as a placeholder in the context position so the name can be filled in when the template is instantiated (e.g. `[Packdown] {Campaign} — Remove campaign landing page`).

### Create & Edit

**IMPORTANT: When creating a task, fill in ALL applicable fields.** Do not create bare tasks with just a name. Ask the user for any information you don't already have. Use the naming conventions above for the `--name` flag. A well-created task should include as many of these as possible:

- `--current` — **preferred**: auto-resolves the active sprint list (no need to know the list ID)
- `--list-id` — explicit list ID (use when creating outside the current sprint)
- `--name` — task name (required — follow naming conventions above)
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

**IMPORTANT: Always use `--current` instead of hardcoding sprint list IDs.** Sprint list IDs change every sprint. Never cache or remember a specific list ID — always let the CLI resolve it dynamically.

After creating a task, consider adding checklists for acceptance criteria or subtasks:

```bash
clickup task checklist add <task-id> "Acceptance Criteria"
clickup task checklist item add <checklist-id> "Unit tests pass"
clickup task checklist item add <checklist-id> "Code reviewed"
```

Example of a well-populated task creation:

```bash
clickup task create --current \
  --name "[Bug] Auth — Fix login timeout on slow connections (API)" \
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

**Tags:** Before creating a task, check what tags are available in the space with `clickup tag list`. Use existing tags rather than inventing new ones — tags should be consistent across the workspace. If no suitable tag exists, discuss with the user before creating a new one.

**Bulk create from JSON file** — use `--from-file` for batch task creation:

```bash
# Create many tasks at once from a JSON file
clickup task create --current --from-file tasks.json
clickup task create --current --from-file tasks.json --json
```

The JSON file should contain an array of task objects:

```json
[
  {
    "name": "Design homepage",
    "description": "Create wireframes and mockups",
    "status": "open",
    "priority": 2,
    "due_date": "2026-03-15",
    "time_estimate": "4h",
    "tags": ["design"],
    "parent": "86abc123",
    "fields": [{"name": "Environment", "value": "staging"}]
  },
  {
    "name": "Implement auth flow",
    "start_date": "2026-03-01",
    "due_date": "2026-03-10",
    "assignees": [12345678],
    "points": 5
  }
]
```

Errors on individual tasks are reported but don't stop the batch. A summary is printed at the end.

```bash
# Edit a task (auto-detects from git branch)
clickup task edit --status "in progress" --priority 2
clickup task edit CU-abc123 --field "Environment=production"
clickup task edit --due-date 2025-03-01 --time-estimate 4h

# Bulk edit multiple tasks at once — pass all IDs as positional args
clickup task edit 86abc1 86abc2 86abc3 --status "Closed"
clickup task edit 86abc1 86abc2 86abc3 --due-date 2026-03-01 --priority 2

# Tags — add without removing existing
clickup task edit CU-abc123 --add-tags new-feature-development
clickup task edit 86abc1 86abc2 --add-tags r&d,new-app-development

# Tags — remove specific tags
clickup task edit CU-abc123 --remove-tags fix

# Tags — replace all (use with caution)
clickup task edit CU-abc123 --tags "ios,android,new-app-development"

# Custom fields
clickup task edit CU-abc123 --field "Environment=production"
clickup task edit CU-abc123 --clear-field "Environment"
clickup field list --list-id 12345    # Discover available fields
```

**Bulk edit workflow — closing all subtasks of a parent:**

1. View the parent task with `--json` to get subtask IDs:
   ```bash
   clickup task view 86parent --json
   # → .subtasks[].id gives you the child IDs
   ```
2. If subtasks themselves have children, view each subtask with `--json` to get the next level of IDs.
3. Pass all collected IDs to a single `task edit`:
   ```bash
   clickup task edit 86child1 86child2 86child3 ... --status "Closed"
   ```
4. Errors on individual tasks are reported but don't stop the batch. A summary line shows `Updated X/N tasks`.

**Important:** Check `clickup status list` or the validation error output to find the correct status name for the space — statuses vary between spaces (e.g., "done" vs "Closed" vs "complete").

### Multi-List (Add/Remove Tasks from Lists)

```bash
# Add a task to a sprint list (task stays in its original list too)
clickup task list-add 86abc123 --list-id 901613544162

# Add multiple tasks to the same list at once
clickup task list-add 86abc1 86abc2 86abc3 --list-id 901613544162

# Remove a task from a list (must belong to at least one other list)
clickup task list-remove 86abc123 --list-id 901613544162

# Remove multiple tasks from the same list
clickup task list-remove 86abc1 86abc2 86abc3 --list-id 901613544162
```

Use `list-add` when a task needs to appear in multiple lists — e.g., a campaign task that also belongs in an engineering sprint. Use `list-remove` to undo this. Neither command moves the task; they manage secondary list memberships.

### Status Management

```bash
# Set status (supports fuzzy matching: "review" matches "code review")
clickup status set "in progress"
clickup status set "in progress" CU-abc123

# List available statuses
clickup status list
clickup status list --space 12345

# Add a new status to a space (inserted before Closed)
clickup status add "done"
clickup status add "QA Review" --color "#7C4DFF"
clickup status add "done" -y              # Skip confirmation
clickup status add "done" --space 12345   # Specific space
```

Status values are fuzzy-matched: exact match > contains match > fuzzy match. If ambiguous, the CLI picks the most specific match and prints a warning.

**Guardrails for `status add`:** Only suggest adding a status when there's a genuine gap in the workflow (e.g., a space has "Closed" but no "done" equivalent). Always confirm with the user before adding. Never remove statuses without explicit user instruction — statuses affect all tasks in the space.

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

# List comments (newest first — .[0] is most recent, .[-1] is oldest)
clickup comment list CU-abc123

# Edit/delete comments (you can only edit comments you authored)
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

**Auto-detection:** `task view` can detect the associated ClickUp task even on branches without task IDs by finding the branch's GitHub PR URL in task descriptions.

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

# Timesheet: list all your time entries for a date range
clickup task time list --start-date 2026-02-01 --end-date 2026-02-28
clickup task time list --start-date 2026-02-01 --end-date 2026-02-28 --json

# Timesheet for a specific user
clickup task time list --start-date 2026-02-01 --end-date 2026-02-28 --assignee 54695018

# Timesheet for all workspace members
clickup task time list --start-date 2026-02-01 --end-date 2026-02-28 --assignee all
```

When `--start-date` and `--end-date` are provided, the command switches to **timesheet mode** — querying all time entries across tasks for the date range, grouped by task. Defaults to the current user; use `--assignee all` for everyone or `--assignee <user-id>` for a specific person.

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

## Docs

Manage ClickUp Docs and their pages via the v3 API. All read commands support `--json`, `--jq`, and `--template`.

> **API limitation:** Delete, archive, and restore operations for Docs and Pages are not available via the public ClickUp v3 API at this time.

### List & View

```bash
# List all Docs in the workspace
clickup doc list
clickup doc list --json

# Filter by parent location
clickup doc list --parent-id 123456 --parent-type SPACE

# Include archived or deleted Docs
clickup doc list --archived
clickup doc list --deleted

# Paginate results
clickup doc list --limit 20 --cursor <cursor>

# View a specific Doc
clickup doc view <doc-id>
clickup doc view <doc-id> --json
```

### Create

```bash
# Create a Doc (creates an initial empty page by default)
clickup doc create --name "Project Runbook"

# Create in a specific space with public visibility
clickup doc create --name "Team Wiki" \
  --parent-id 123456 --parent-type SPACE \
  --visibility PUBLIC

# Create without an initial page
clickup doc create --name "Drafts" --create-page=false
```

### Pages

```bash
# List pages in a Doc (returns a tree structure)
clickup doc page list <doc-id>
clickup doc page list <doc-id> --max-depth 0   # Top-level only
clickup doc page list <doc-id> --json

# View a page (with optional content format)
clickup doc page view <doc-id> <page-id>
clickup doc page view <doc-id> <page-id> --content-format text/md
clickup doc page view <doc-id> <page-id> --json

# Create a page
clickup doc page create <doc-id> --name "Introduction"
clickup doc page create <doc-id> --name "Setup Guide" \
  --content "# Setup\n\nFollow these steps..." \
  --content-format text/md

# Create a nested page
clickup doc page create <doc-id> --name "Advanced Config" \
  --parent-page-id <parent-page-id>

# Edit a page (replace, append, or prepend content)
clickup doc page edit <doc-id> <page-id> --content "New content"
clickup doc page edit <doc-id> <page-id> \
  --content "## Release Notes\n\n- Fixed bug X" \
  --content-edit-mode append
clickup doc page edit <doc-id> <page-id> --name "Updated Title"
```

### JSON-first agent usage

```bash
# Get all doc IDs
clickup doc list --jq '.[].id'

# Get page IDs for a specific doc
clickup doc page list <doc-id> --jq '.pages[].id'

# Append release notes programmatically
clickup doc page edit <doc-id> <page-id> \
  --content "## v1.2.3\n\n- Fixed auth timeout" \
  --content-edit-mode append \
  --content-format text/md

# Create doc and capture its ID
clickup doc create --name "Sprint Retro" --json | jq -r '.id'
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
- **Bulk operations**: `task create --from-file` creates many tasks from JSON; `task edit ID1 ID2 ...` applies the same changes to multiple tasks. To bulk-edit subtasks: view the parent with `--json`, extract subtask IDs from `.subtasks[].id`, then pass them all to `task edit`
- **Subtask visibility**: `task view` shows subtask due/start dates inline, so you can spot-check deadlines without viewing each subtask individually
- **Multi-list**: `task list-add`/`task list-remove` manage secondary list memberships — useful for cross-team sprint planning
- **Naming conventions**: Task names follow `[Work Type] Context — Action (Platform)` format for sprint-board scannability. Check existing tasks in the list for the prevailing convention before creating
- **Tag reuse**: Always check available tags with `clickup tag list` before creating tasks. Use existing tags for consistency; don't invent new ones without user confirmation
