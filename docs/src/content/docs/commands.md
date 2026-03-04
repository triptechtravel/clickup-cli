---
title: Command reference
description: Overview of all clickup CLI commands, grouped by category.
---

# Command reference

All commands are invoked as subcommands of `clickup`. Run `clickup --help` for a summary, or `clickup <command> --help` for details on any command.

Every command that produces output supports `--json` for machine-readable output, `--jq` for inline filtering, and `--template` for Go template formatting.

Flag details, examples, and options for each command are in the auto-generated [Reference](/clickup-cli/reference/clickup/) pages.

---

## Task management

| Command | Description |
|---------|-------------|
| [`task view`](/clickup-cli/reference/clickup_task_view/) | View task details (auto-detects from git branch) |
| [`task list`](/clickup-cli/reference/clickup_task_list/) | List tasks in a ClickUp list with filters |
| [`task create`](/clickup-cli/reference/clickup_task_create/) | Create a new task (interactive or flags, supports `--from-file` for bulk) |
| [`task edit`](/clickup-cli/reference/clickup_task_edit/) | Edit task fields — supports bulk edit with multiple task IDs |
| [`task delete`](/clickup-cli/reference/clickup_task_delete/) | Delete a task permanently |
| [`task search`](/clickup-cli/reference/clickup_task_search/) | Search tasks with fuzzy matching |
| [`task recent`](/clickup-cli/reference/clickup_task_recent/) | Show recently updated tasks with folder/list context |
| [`task activity`](/clickup-cli/reference/clickup_task_activity/) | View task details and comment history |
| [`task list-add`](/clickup-cli/reference/clickup_task_list-add/) | Add tasks to an additional list |
| [`task list-remove`](/clickup-cli/reference/clickup_task_list-remove/) | Remove tasks from a list |

### Time tracking

| Command | Description |
|---------|-------------|
| [`task time log`](/clickup-cli/reference/clickup_task_time_log/) | Log time to a task |
| [`task time list`](/clickup-cli/reference/clickup_task_time_list/) | View time entries (per-task or timesheet mode with date ranges) |
| [`task time delete`](/clickup-cli/reference/clickup_task_time_delete/) | Delete a time entry |

### Dependencies & checklists

| Command | Description |
|---------|-------------|
| [`task dependency add`](/clickup-cli/reference/clickup_task_dependency_add/) | Add a dependency (depends-on or blocks) |
| [`task dependency remove`](/clickup-cli/reference/clickup_task_dependency_remove/) | Remove a dependency |
| [`task checklist add`](/clickup-cli/reference/clickup_task_checklist_add/) | Create a checklist on a task |
| [`task checklist remove`](/clickup-cli/reference/clickup_task_checklist_remove/) | Delete a checklist |
| [`task checklist item add`](/clickup-cli/reference/clickup_task_checklist_item_add/) | Add an item to a checklist |
| [`task checklist item resolve`](/clickup-cli/reference/clickup_task_checklist_item_resolve/) | Mark a checklist item as resolved |
| [`task checklist item remove`](/clickup-cli/reference/clickup_task_checklist_item_remove/) | Remove a checklist item |

---

## Status & fields

| Command | Description |
|---------|-------------|
| [`status set`](/clickup-cli/reference/clickup_status_set/) | Change task status with fuzzy matching |
| [`status list`](/clickup-cli/reference/clickup_status_list/) | List available statuses for a space |
| [`status add`](/clickup-cli/reference/clickup_status_add/) | Add a new status to a space |
| [`field list`](/clickup-cli/reference/clickup_field_list/) | List custom fields for a list |
| [`tag list`](/clickup-cli/reference/clickup_tag_list/) | List available tags for a space |

---

## Comments

| Command | Description |
|---------|-------------|
| [`comment add`](/clickup-cli/reference/clickup_comment_add/) | Add a comment (supports @mentions) |
| [`comment list`](/clickup-cli/reference/clickup_comment_list/) | List comments on a task |
| [`comment edit`](/clickup-cli/reference/clickup_comment_edit/) | Edit an existing comment |
| [`comment delete`](/clickup-cli/reference/clickup_comment_delete/) | Delete a comment |

---

## Git & GitHub integration

| Command | Description |
|---------|-------------|
| [`link pr`](/clickup-cli/reference/clickup_link_pr/) | Link a GitHub PR to a ClickUp task |
| [`link sync`](/clickup-cli/reference/clickup_link_sync/) | Sync ClickUp task info into a GitHub PR body (and link back) |
| [`link branch`](/clickup-cli/reference/clickup_link_branch/) | Link the current git branch to a task |
| [`link commit`](/clickup-cli/reference/clickup_link_commit/) | Link a git commit to a task |

---

## Sprints

| Command | Description |
|---------|-------------|
| [`sprint current`](/clickup-cli/reference/clickup_sprint_current/) | Show tasks in the active sprint, grouped by status |
| [`sprint list`](/clickup-cli/reference/clickup_sprint_list/) | List sprints in the sprint folder |

---

## Workspace

| Command | Description |
|---------|-------------|
| [`inbox`](/clickup-cli/reference/clickup_inbox/) | Show recent @mentions across your workspace |
| [`member list`](/clickup-cli/reference/clickup_member_list/) | List workspace members with IDs, usernames, emails, and roles |
| [`space list`](/clickup-cli/reference/clickup_space_list/) | List spaces in your workspace |
| [`space select`](/clickup-cli/reference/clickup_space_select/) | Interactively select a default space |

---

## Setup & utilities

| Command | Description |
|---------|-------------|
| [`auth login`](/clickup-cli/reference/clickup_auth_login/) | Authenticate (token prompt, `--oauth`, or `--with-token` for CI) |
| [`auth logout`](/clickup-cli/reference/clickup_auth_logout/) | Remove stored credentials |
| [`auth status`](/clickup-cli/reference/clickup_auth_status/) | Show current authentication state |
| [`version`](/clickup-cli/reference/clickup_version/) | Print version, commit, and build date |
| [`completion`](/clickup-cli/reference/clickup_completion/) | Generate shell completion scripts |
