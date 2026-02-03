---
title: Getting started
description: Initial setup walkthrough for the clickup CLI.
---

# Getting started

This guide walks through initial setup and your first interaction with the CLI.

## Step 1: Authenticate

Run the login command to store your ClickUp API token:

```sh
clickup auth login
```

You will be prompted to paste a personal API token. To generate one, go to **ClickUp > Settings > ClickUp API > API tokens**.

The login flow also selects your default workspace. If your account belongs to multiple workspaces, you will be asked to choose one.

### Alternative authentication methods

**OAuth browser flow** (requires a registered OAuth app):

```sh
clickup auth login --oauth
```

**CI / non-interactive mode** (pipe a token via stdin):

```sh
echo "$CLICKUP_TOKEN" | clickup auth login --with-token
```

### Check authentication status

```sh
clickup auth status
```

## Step 2: Select a space

After logging in, select the default ClickUp space for your commands:

```sh
clickup space select
```

This presents an interactive list of spaces in your workspace. The selection is saved to `~/.config/clickup/config.yml`.

To see all available spaces:

```sh
clickup space list
```

## Step 3: View a task

If your git branch contains a ClickUp task ID, the CLI detects it automatically:

```sh
# On branch feature/CU-ae27de-add-user-auth
clickup task view
```

Or specify a task ID directly:

```sh
clickup task view CU-abc123
```

The output includes the task name, status, priority, assignees, watchers, tags, dates, points, time tracking, location, dependencies, checklists, custom fields, URL, and description.

## Step 4: Change a status

Set a task's status using fuzzy matching. The CLI matches your input against the available statuses for the task's space:

```sh
clickup status set "in progress"
```

You can use partial input and the CLI will find the closest match:

```sh
clickup status set "prog"
```

## Step 5: Link a pull request

After opening a GitHub PR on a branch that contains a task ID, link it to the ClickUp task:

```sh
clickup link pr
```

This stores a link to the GitHub PR on the ClickUp task. By default, links are stored in a managed section of the task description. You can optionally configure a custom field for link storage (see [Configuration](/clickup-cli/configuration/#github-link-storage)).

Requires the [GitHub CLI](https://cli.github.com/) (`gh`) to be installed and authenticated.

## Next steps

- See the full [Command reference](/clickup-cli/commands/) for all available commands and flags.
- Read about [Configuration](/clickup-cli/configuration/) to customize the CLI for your workflow.
- Learn how [Git integration](/clickup-cli/git-integration/) detects task IDs from branch names.
