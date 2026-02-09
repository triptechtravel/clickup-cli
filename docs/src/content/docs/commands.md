---
title: Command reference
description: Complete reference for all clickup CLI commands and flags.
---

# Command reference

All commands are invoked as subcommands of `clickup`. Run `clickup --help` for a summary, or `clickup <command> --help` for details on any command.

Every command that produces output supports `--json` for machine-readable output and `--jq` for inline filtering. After successful operations, the CLI prints a quick-actions footer suggesting contextual next steps — use `--json` to suppress these footers for scripting.

---

## auth

Manage authentication credentials.

### `auth login`

Authenticate with a ClickUp account. By default, prompts for a personal API token. The token is validated against the ClickUp API and stored in the system keyring. The login flow also selects a default workspace.

```sh
# Interactive token entry (default)
clickup auth login

# Pipe token for CI
echo "pk_12345" | clickup auth login --with-token

# Use OAuth browser flow
clickup auth login --oauth
```

| Flag | Description |
|------|-------------|
| `--with-token` | Read token from standard input (for CI environments) |
| `--oauth` | Use OAuth browser flow (requires a registered OAuth app) |

### `auth logout`

Remove stored credentials from the system keyring.

```sh
clickup auth logout
```

### `auth status`

Display the current authentication state, including the logged-in user and workspace.

```sh
clickup auth status
```

---

## task

View, list, create, and edit ClickUp tasks.

### `task view [TASK-ID]`

Display detailed information about a single task, including name, status, priority, assignees, watchers, tags, dates, points, time estimate, time spent, URL, description, custom fields, dependencies, linked tasks, checklists, and attachments.

If no task ID is provided, the CLI auto-detects it from the current git branch name.

```sh
# Auto-detect task from git branch
clickup task view

# View a specific task
clickup task view 86a3xrwkp

# Output as JSON
clickup task view 86a3xrwkp --json
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `task list`

List tasks from a ClickUp list with optional filters. Displays a table with ID, name, status, priority, assignee, tags, and due date columns.

```sh
# List tasks in a ClickUp list
clickup task list --list-id 12345

# Filter by assignee and status
clickup task list --list-id 12345 --assignee me --status "in progress"
```

| Flag | Description |
|------|-------------|
| `--list-id ID` | ClickUp list ID (required) |
| `--assignee ID` | Filter by assignee ID(s) (repeatable) |
| `--status STATUS` | Filter by status(es) (repeatable) |
| `--sprint NAME` | Filter by sprint name |
| `--page N` | Page number for pagination (starts at 0) |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `task create`

Create a new task in a ClickUp list. If `--name` is not provided, the command enters interactive mode and prompts for the task name, description, status, and priority. Supports setting tags, due dates, start dates, time estimates, sprint points, custom fields, parent tasks, and task types at creation time.

When `--status` is provided, the CLI validates it against the available statuses for the list's space using fuzzy matching (same strategy as `status set`). If the provided status fuzzy-matches a valid status, a warning is shown (e.g., `Status "prog" matched to "in progress"`). If no match is found, an error is returned listing all available statuses.

```sh
# Create with flags
clickup task create --list-id 12345 --name "Fix login bug" --priority 2

# Interactive mode
clickup task create --list-id 12345

# Create with due date, points, and custom field
clickup task create --list-id 12345 --name "Fix auth" --priority 2 --due-date 2025-02-14 --points 3 --field "Environment=production"

# Create a subtask
clickup task create --list-id 12345 --name "Subtask" --parent 86abc123

# Create a milestone
clickup task create --list-id 12345 --name "v2.0 Release" --type 1
```

| Flag | Description |
|------|-------------|
| `--list-id ID` | ClickUp list ID (required) |
| `--name TEXT` | Task name |
| `--description TEXT` | Task description |
| `--markdown-description TEXT` | Task description in markdown |
| `--status STATUS` | Task status |
| `--priority N` | Priority: 1=Urgent, 2=High, 3=Normal, 4=Low |
| `--assignee ID` | Assignee user ID(s) (repeatable) |
| `--tags TAG` | Tags to add (repeatable) |
| `--due-date DATE` | Due date (YYYY-MM-DD) |
| `--start-date DATE` | Start date (YYYY-MM-DD) |
| `--due-date-time` | Include time component in due date |
| `--start-date-time` | Include time component in start date |
| `--time-estimate DUR` | Time estimate (e.g. "2h", "30m", "1h30m") |
| `--points N` | Sprint/story points |
| `--parent ID` | Parent task ID (create as subtask) |
| `--links-to ID` | Link to another task by ID |
| `--type N` | Task type (0=task, 1=milestone, or custom type ID) |
| `--notify-all` | Notify all assignees and watchers |
| `--field "Name=value"` | Set a custom field value (repeatable) |
| `--json` | Output created task as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `task edit [TASK-ID]`

Edit fields on an existing task. At least one field flag must be provided. If no task ID is given, it is auto-detected from the current git branch.

When `--status` is provided, the CLI validates it against available statuses for the task's space using fuzzy matching. A warning is shown if the match is not exact (e.g., `Status "prog" matched to "in progress"`). An error is returned if no match is found.

```sh
# Edit the task detected from the current branch
clickup task edit --name "Updated title" --priority 2

# Edit a specific task
clickup task edit CU-abc123 --status "in review"

# Set due date and time estimate
clickup task edit --due-date 2025-02-14 --time-estimate 4h

# Remove an assignee
clickup task edit CU-abc123 --remove-assignee 12345

# Set sprint points
clickup task edit --points 5

# Set a custom field
clickup task edit CU-abc123 --field "Environment=production"

# Clear a custom field
clickup task edit CU-abc123 --clear-field "Environment"

# Reparent a task (make it a subtask)
clickup task edit CU-abc123 --parent 86def456

# Change task type to milestone
clickup task edit CU-abc123 --type 1
```

| Flag | Description |
|------|-------------|
| `--name TEXT` | New task name |
| `--description TEXT` | New task description |
| `--markdown-description TEXT` | New task description in markdown |
| `--status STATUS` | New task status |
| `--priority N` | New priority: 1=Urgent, 2=High, 3=Normal, 4=Low |
| `--assignee ID` | Assignee user ID(s) to add (repeatable) |
| `--remove-assignee ID` | Assignee user ID(s) to remove (repeatable) |
| `--tags TAG` | Set tags (replaces existing, repeatable) |
| `--due-date DATE` | Due date (YYYY-MM-DD, "none" to clear) |
| `--start-date DATE` | Start date (YYYY-MM-DD, "none" to clear) |
| `--due-date-time` | Include time component in due date |
| `--start-date-time` | Include time component in start date |
| `--time-estimate DUR` | Time estimate (e.g. "2h", "30m"; "0" to clear) |
| `--points N` | Sprint/story points (-1 to clear) |
| `--parent ID` | Parent task ID (reparent task) |
| `--links-to ID` | Link to another task by ID |
| `--type N` | Task type (0=task, 1=milestone, or custom type ID) |
| `--notify-all` | Notify all assignees and watchers |
| `--field "Name=value"` | Set a custom field value (repeatable) |
| `--clear-field "Name"` | Clear a custom field value (repeatable) |
| `--json` | Output updated task as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `task search <query>`

Search tasks by name with fuzzy matching and optional comment search. The `<query>` argument matches against task names as a case-insensitive substring. To search by task ID, pass the full ID (e.g., `CU-abc123`).

If no results are found, the CLI suggests running `clickup task recent` or `clickup sprint current` to discover active work. In interactive mode, "Show my recent tasks" is offered as a menu option.

When no exact substring matches are found but fuzzy results exist, a message indicates the results are fuzzy. Use `--exact` to suppress fuzzy matches and only show exact substring results.

```sh
# Search by name substring
clickup task search "geozone" --comments --pick

# Search within a specific folder (use 'task recent' to discover folders)
clickup task search "geozone" --folder "Engineering Sprint"

# Only show exact substring matches (no fuzzy results)
clickup task search "FAQ" --exact

# Search and output as JSON
clickup task search "login bug" --json
```

| Flag | Description |
|------|-------------|
| `--space NAME_OR_ID` | Limit search to a specific space (name or ID) |
| `--folder NAME` | Limit search to a specific folder (name, substring match) |
| `--pick` | Interactively pick from results and print its task ID |
| `--comments` | Also search within task comment bodies |
| `--exact` | Only show exact substring matches (no fuzzy results) |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `task recent`

Show recently updated tasks with folder and list context. By default shows tasks assigned to you. Use `--all` for all team activity. Tasks from archived folders are automatically filtered out.

This command is particularly useful for discovering which folders and lists contain active work, so you can narrow searches with `--folder` or `--space`.

```sh
# Show your recent tasks with location context
clickup task recent

# Show all team activity
clickup task recent --all

# Only show tasks from sprint folders
clickup task recent --sprint

# JSON output for scripting or AI agents
clickup task recent --json --limit 10
```

| Flag | Description |
|------|-------------|
| `--limit N` | Maximum number of tasks to show (default 20) |
| `--all` | Show all team tasks, not just yours |
| `--sprint` | Only show tasks from sprint folders |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `task activity [TASK-ID]`

View a task's details and chronological comment history.

```sh
clickup task activity 86a3xrwkp
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `task time log [TASK-ID]`

Log a time entry against a task.

```sh
clickup task time log --duration 2h --description "Implemented auth"
```

| Flag | Description |
|------|-------------|
| `--duration DUR` | Duration to log (required, e.g. "2h", "30m", "1h30m") |
| `--description TEXT` | Description of the work performed |
| `--date DATE` | Date for the time entry (defaults to today) |
| `--billable` | Mark the time entry as billable |

### `task time list [TASK-ID]`

View time entries for a task.

```sh
clickup task time list 86a3xrwkp
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `task dependency add <TASK-ID>`

Add a dependency relationship between two tasks. Use `--depends-on` to indicate a task waits on another, or `--blocks` to indicate a task blocks another.

```sh
# This task depends on (waits for) another task
clickup task dependency add 86abc123 --depends-on 86def456

# This task blocks another task
clickup task dependency add 86abc123 --blocks 86def456
```

| Flag | Description |
|------|-------------|
| `--depends-on ID` | Task ID that this task depends on (waits for) |
| `--blocks ID` | Task ID that this task blocks |

### `task dependency remove <TASK-ID>`

Remove a dependency relationship between two tasks.

```sh
clickup task dependency remove 86abc123 --depends-on 86def456
clickup task dependency remove 86abc123 --blocks 86def456
```

| Flag | Description |
|------|-------------|
| `--depends-on ID` | Task ID to remove depends-on relationship with |
| `--blocks ID` | Task ID to remove blocks relationship with |

### `task checklist add <TASK-ID> <NAME>`

Create a checklist on a task.

```sh
clickup task checklist add 86abc123 "Deploy Steps"
```

### `task checklist remove <CHECKLIST-ID>`

Delete a checklist.

```sh
clickup task checklist remove b955c4dc-b8ee-4488-b0c1-example
```

### `task checklist item add <CHECKLIST-ID> <NAME>`

Add an item to a checklist.

```sh
clickup task checklist item add b955c4dc-example "Run migrations"
```

### `task checklist item resolve <CHECKLIST-ID> <ITEM-ID>`

Mark a checklist item as resolved.

```sh
clickup task checklist item resolve b955c4dc-example 21e08dc8-example
```

### `task checklist item remove <CHECKLIST-ID> <ITEM-ID>`

Remove an item from a checklist.

```sh
clickup task checklist item remove b955c4dc-example 21e08dc8-example
```

---

## member

Manage workspace members.

### `member list`

List all members in the configured ClickUp workspace. Displays each member's ID, username, email, and role. Member IDs are useful for assigning tasks, adding watchers, and tagging users.

```sh
# List workspace members
clickup member list

# JSON output for scripting
clickup member list --json

# Get a specific member's ID
clickup member list --json --jq '.[] | select(.username == "Isaac") | .id'
```

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

---

## field

Discover custom fields available in your workspace.

### `field list`

List all accessible custom fields for a ClickUp list, showing field name, type, ID, and options (for dropdown/label fields).

```sh
# List fields for a list
clickup field list --list-id 12345

# JSON output
clickup field list --list-id 12345 --json
```

| Flag | Description |
|------|-------------|
| `--list-id ID` | ClickUp list ID (required) |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

---

## comment

Manage comments on ClickUp tasks.

### `comment add [TASK] [BODY]`

Add a comment to a task. If `TASK` is not provided, the task ID is auto-detected from the current git branch. If `BODY` is not provided (or `--editor` is used), your configured editor opens for composing the comment.

**@mentions:** Use `@username` in the comment body to mention workspace members. Usernames are resolved against your workspace's member list (same as `clickup member list`) using case-insensitive matching. Resolved mentions become real ClickUp @mentions that trigger notifications. Unresolved `@` text is left as-is.

```sh
# Add a comment to the branch's task
clickup comment add "" "Fixed the login bug"

# Add a comment to a specific task
clickup comment add abc123 "Deployed to staging"

# Mention a teammate
clickup comment add abc123 "Hey @Isaac can you review this?"

# Open your editor to compose the comment
clickup comment add --editor
```

| Flag | Description |
|------|-------------|
| `-e`, `--editor` | Open editor to compose comment body |

### `comment list [TASK-ID]`

List comments on a task.

```sh
clickup comment list CU-abc123
```

### `comment edit <COMMENT_ID> [BODY]`

Edit an existing comment. If `BODY` is not provided (or `--editor` is used), your configured editor opens for editing.

```sh
# Edit a comment by ID
clickup comment edit 12345 "Updated comment text"

# Open editor to edit the comment
clickup comment edit 12345 --editor
```

| Flag | Description |
|------|-------------|
| `-e`, `--editor` | Open editor to compose comment body |

### `comment delete <COMMENT_ID>`

Delete a comment from a ClickUp task. Find comment IDs with `comment list TASK_ID --json`.

```sh
# Delete a comment (with confirmation prompt)
clickup comment delete 90160162431205

# Delete without confirmation
clickup comment delete 90160162431205 --yes
```

| Flag | Description |
|------|-------------|
| `-y`, `--yes` | Skip confirmation prompt |

---

## status

Manage task statuses.

### `status set <STATUS> [TASK]`

Change a task's status using fuzzy matching. The `STATUS` argument is matched against available statuses for the task's space using a three-tier strategy:

1. Exact match (case-insensitive)
2. Contains match (case-insensitive)
3. Fuzzy match using normalized fold ranking

If `TASK` is not provided, the task ID is auto-detected from the current git branch.

```sh
# Set status using auto-detected task from branch
clickup status set "in progress"

# Set status for a specific task
clickup status set "done" CU-abc123

# Fuzzy matching works too
clickup status set "prog" CU-abc123
```

### `status list`

List all available statuses for the configured space.

```sh
clickup status list

# List statuses for a specific space
clickup status list --space 12345
```

| Flag | Description |
|------|-------------|
| `--space ID` | Space ID to list statuses for (defaults to configured space) |

---

## link

Link GitHub artifacts to ClickUp tasks. All link commands update the task idempotently -- links are stored in a managed section of the task description using ClickUp's `markdown_description` API field, so they render as rich text with clickable links, bold formatting, and code blocks directly in the ClickUp UI.

### `link pr [NUMBER]`

Link a GitHub pull request to a ClickUp task. If `NUMBER` is not provided, the current PR is detected using the GitHub CLI (`gh`). The ClickUp task ID is auto-detected from the current git branch. The link is stored idempotently -- running the command again updates the existing entry rather than creating a duplicate.

Requires the [GitHub CLI](https://cli.github.com/) to be installed and authenticated.

```sh
# Link the current branch's PR to the detected task
clickup link pr

# Link a specific PR by number
clickup link pr 42

# Link a PR from any repo to any task
clickup link pr 42 --repo owner/repo --task CU-abc123
```

| Flag | Description |
|------|-------------|
| `--task ID` | Target task ID (overrides auto-detection from branch) |
| `--repo OWNER/REPO` | Target GitHub repository (overrides auto-detection) |

### `link sync [NUMBER]`

Sync ClickUp task info into a GitHub PR description and link back to the task. Inserts or updates a table in the PR body showing the task name, status, priority, and assignees. The update is idempotent -- running multiple times replaces the existing block rather than duplicating it.

```sh
# Sync the current PR
clickup link sync

# Sync a specific PR
clickup link sync 42

# Sync from any repo to any task
clickup link sync 42 --repo owner/repo --task CU-abc123
```

| Flag | Description |
|------|-------------|
| `--task ID` | Target task ID (overrides auto-detection from branch) |
| `--repo OWNER/REPO` | Target GitHub repository (overrides auto-detection) |

### `link branch`

Link the current git branch to a ClickUp task. Stores the branch name and repository as a link entry on the task.

```sh
clickup link branch

# Link to a specific task
clickup link branch --task CU-abc123
```

| Flag | Description |
|------|-------------|
| `--task ID` | Target task ID (overrides auto-detection from branch) |

### `link commit [SHA]`

Link a git commit to a ClickUp task. Stores the commit SHA, message, and GitHub URL as a link entry on the task. If `SHA` is not provided, the HEAD commit is used.

```sh
# Link the latest commit
clickup link commit

# Link a specific commit
clickup link commit a1b2c3d

# Link to a specific task
clickup link commit a1b2c3d --task CU-abc123 --repo owner/repo
```

| Flag | Description |
|------|-------------|
| `--task ID` | Target task ID (overrides auto-detection from branch) |
| `--repo OWNER/REPO` | Target GitHub repository (overrides auto-detection) |

---

## sprint

View and manage sprints. Sprints in ClickUp are organized as lists within a folder.

### `sprint current`

Show tasks in the currently active sprint, grouped by status. Displays assignees, priorities, and due dates. The active sprint is determined by finding the list whose start and due dates contain today.

```sh
# Show current sprint tasks
clickup sprint current

# Specify a sprint folder
clickup sprint current --folder 132693664

# JSON output
clickup sprint current --json
```

| Flag | Description |
|------|-------------|
| `--folder ID` | Sprint folder ID (auto-detected if not set) |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

### `sprint list`

List all sprints in the sprint folder, showing ID, name, status (complete, in progress, upcoming), date range, and task count.

```sh
# List all sprints
clickup sprint list

# Specify sprint folder
clickup sprint list --folder 132693664
```

| Flag | Description |
|------|-------------|
| `--folder ID` | Sprint folder ID (auto-detected if not set) |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

---

## space

Manage ClickUp spaces.

### `space list`

List all spaces in the current workspace.

```sh
clickup space list
```

### `space select [NAME]`

Interactively select a default space. If `NAME` is provided, the CLI selects the first space matching that name (useful for scripting). The selection is saved to the config file.

```sh
# Interactive selection
clickup space select

# Select by name
clickup space select "Engineering"

# Save as a directory default
clickup space select --directory
```

| Flag | Description |
|------|-------------|
| `--directory` | Save the selection as a directory default for the current working directory |

---

## inbox

### `inbox`

Show recent @mentions across your workspace. Scans recently updated tasks for mentions in both comments and task descriptions.

```sh
# Show mentions from the last 7 days (default)
clickup inbox

# Look back 30 days
clickup inbox --days 30

# Scan more tasks in a busy workspace
clickup inbox --limit 500

# JSON output for scripting
clickup inbox --json
```

| Flag | Description |
|------|-------------|
| `--days N` | How many days back to search (default: 7) |
| `--limit N` | Maximum number of tasks to scan (default: 200) |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

The command checks both task description text and comment bodies for `@username` mentions. Description mentions show as "mentioned you in description of" to distinguish them from comment mentions. Since ClickUp does not provide a public inbox API, this approximates your inbox by scanning task data directly.

---

## Utility commands

### `version`

Print the CLI version, commit SHA, and build date.

```sh
clickup version
```

### `completion <SHELL>`

Generate shell completion scripts. Supported shells: `bash`, `zsh`, `fish`, `powershell`.

```sh
clickup completion bash
clickup completion zsh
clickup completion fish
clickup completion powershell
```

See the [Installation](/clickup-cli/installation/#shell-completions) page for setup instructions.
