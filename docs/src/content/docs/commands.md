---
title: Command reference
description: Complete reference for all clickup CLI commands and flags.
---

# Command reference

All commands are invoked as subcommands of `clickup`. Run `clickup --help` for a summary, or `clickup <command> --help` for details on any command.

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

Display detailed information about a single task, including name, status, priority, assignees, tags, dates, points, time estimate, time spent, start date, URL, and description.

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

Create a new task in a ClickUp list. If `--name` is not provided, the command enters interactive mode and prompts for the task name, description, status, and priority. Supports setting tags, due dates, start dates, time estimates, and sprint points at creation time.

```sh
# Create with flags
clickup task create --list-id 12345 --name "Fix login bug" --priority 2

# Interactive mode
clickup task create --list-id 12345

# Create with due date and points
clickup task create --list-id 12345 --name "Fix auth" --priority 2 --due-date 2025-02-14 --points 3
```

| Flag | Description |
|------|-------------|
| `--list-id ID` | ClickUp list ID (required) |
| `--name TEXT` | Task name |
| `--description TEXT` | Task description |
| `--status STATUS` | Task status |
| `--priority N` | Priority: 1=Urgent, 2=High, 3=Normal, 4=Low |
| `--assignee ID` | Assignee user ID(s) (repeatable) |
| `--tags TAG` | Tags to add (repeatable) |
| `--due-date DATE` | Due date (YYYY-MM-DD) |
| `--start-date DATE` | Start date (YYYY-MM-DD) |
| `--time-estimate DUR` | Time estimate (e.g. "2h", "30m", "1h30m") |
| `--points N` | Sprint/story points |

### `task edit [TASK-ID]`

Edit fields on an existing task. At least one field flag must be provided. If no task ID is given, it is auto-detected from the current git branch.

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
```

| Flag | Description |
|------|-------------|
| `--name TEXT` | New task name |
| `--description TEXT` | New task description |
| `--status STATUS` | New task status |
| `--priority N` | New priority: 1=Urgent, 2=High, 3=Normal, 4=Low |
| `--assignee ID` | Assignee user ID(s) to set (repeatable) |
| `--remove-assignee ID` | Assignee user ID(s) to remove (repeatable) |
| `--tags TAG` | Set tags (replaces existing, repeatable) |
| `--due-date DATE` | Due date (YYYY-MM-DD, "none" to clear) |
| `--start-date DATE` | Start date (YYYY-MM-DD, "none" to clear) |
| `--time-estimate DUR` | Time estimate (e.g. "2h", "30m"; "0" to clear) |
| `--points N` | Sprint/story points (-1 to clear) |

### `task search <query>`

Search tasks by name with fuzzy matching and optional comment search. The `<query>` argument matches against task names as a case-insensitive substring. To search by task ID, pass the full ID (e.g., `CU-abc123`).

```sh
# Search by name substring
clickup task search "geozone" --comments --pick

# Search and output as JSON
clickup task search "login bug" --json
```

| Flag | Description |
|------|-------------|
| `--space ID` | Limit search to a specific space (defaults to configured space) |
| `--folder ID` | Limit search to a specific folder |
| `--pick` | Interactively pick from results and view the selected task |
| `--comments` | Also search within task comment bodies |
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

---

## comment

Manage comments on ClickUp tasks.

### `comment add [TASK] [BODY]`

Add a comment to a task. If `TASK` is not provided, the task ID is auto-detected from the current git branch. If `BODY` is not provided (or `--editor` is used), your configured editor opens for composing the comment.

```sh
# Add a comment to the branch's task
clickup comment add "" "Fixed the login bug"

# Add a comment to a specific task
clickup comment add abc123 "Deployed to staging"

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

Link GitHub artifacts to ClickUp tasks. All link commands post a comment on the ClickUp task with a formatted link.

### `link pr [NUMBER]`

Link a GitHub pull request to a ClickUp task. If `NUMBER` is not provided, the current PR is detected using the GitHub CLI (`gh`). The ClickUp task ID is auto-detected from the current git branch.

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

Link the current git branch to a ClickUp task by posting a comment with the branch name and repository.

```sh
clickup link branch

# Link to a specific task
clickup link branch --task CU-abc123 --repo owner/repo
```

| Flag | Description |
|------|-------------|
| `--task ID` | Target task ID (overrides auto-detection from branch) |
| `--repo OWNER/REPO` | Target GitHub repository (overrides auto-detection) |

### `link commit [SHA]`

Link a git commit to a ClickUp task by posting a comment with the commit SHA, message, and a link to the commit on GitHub. If `SHA` is not provided, the HEAD commit is used.

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

Show recent comments that @mention you across your workspace. Scans recently updated tasks for comments containing your username.

```sh
# Show mentions from the last 7 days (default)
clickup inbox

# Look back 30 days
clickup inbox --days 30

# JSON output for scripting
clickup inbox --json
```

| Flag | Description |
|------|-------------|
| `--days N` | How many days back to search (default: 7) |
| `--limit N` | Maximum number of tasks to scan (default: 50) |
| `--json` | Output as JSON |
| `--jq EXPR` | Filter JSON output with a jq expression |

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
