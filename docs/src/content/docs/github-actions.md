---
title: GitHub Actions
description: Pre-built GitHub Actions workflows for automating ClickUp task updates on PR events.
---

# GitHub Actions

Automate ClickUp task updates from GitHub PR events. The CLI provides all the building blocks -- the workflows below wire them to GitHub events.

Copy these from the [`examples/`](https://github.com/triptechtravel/clickup-cli/tree/main/examples) directory in the repository.

## Prerequisites

Add your ClickUp API token as a repository secret named `CLICKUP_TOKEN`:

1. Go to your repository **Settings > Secrets and variables > Actions**
2. Click **New repository secret**
3. Name: `CLICKUP_TOKEN`, Value: your ClickUp personal API token

## Sync task info on PR open

Updates the GitHub PR description with a table showing the ClickUp task name, status, priority, and assignees. Also posts a link comment on the ClickUp task.

The update is idempotent -- running multiple times replaces the existing block rather than duplicating it.

```yaml
name: ClickUp Sync
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  sync:
    runs-on: ubuntu-latest
    permissions:
      pull-requests: write
    steps:
      - uses: actions/checkout@v4
      - name: Install clickup CLI
        run: |
          curl -sL https://github.com/triptechtravel/clickup-cli/releases/latest/download/clickup_linux_amd64.tar.gz | tar xz
          sudo mv clickup /usr/local/bin/
      - name: Authenticate
        run: echo "${{ "{{" }} secrets.CLICKUP_TOKEN {{ "}}" }}" | clickup auth login --with-token
      - name: Sync PR with ClickUp task
        run: clickup link sync ${{ "{{" }} github.event.pull_request.number {{ "}}" }}
        env:
          GH_TOKEN: ${{ "{{" }} secrets.GITHUB_TOKEN {{ "}}" }}
```

## Set status to "done" on merge

Automatically marks the ClickUp task as done when the associated PR is merged.

```yaml
name: ClickUp Done
on:
  pull_request:
    types: [closed]

jobs:
  done:
    if: github.event.pull_request.merged == true
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ "{{" }} github.event.pull_request.head.ref {{ "}}" }}
      - name: Install clickup CLI
        run: |
          curl -sL https://github.com/triptechtravel/clickup-cli/releases/latest/download/clickup_linux_amd64.tar.gz | tar xz
          sudo mv clickup /usr/local/bin/
      - name: Authenticate
        run: echo "${{ "{{" }} secrets.CLICKUP_TOKEN {{ "}}" }}" | clickup auth login --with-token
      - name: Set task to done
        run: clickup status set "done"
```

## Update status on PR review

Changes ClickUp task status based on review decisions:
- **Approved** -- moves to "awaiting testing"
- **Changes requested** -- moves back to "in progress" and posts a comment

Adjust the status names to match your ClickUp workflow.

```yaml
name: ClickUp Review Status
on:
  pull_request_review:
    types: [submitted]

jobs:
  update-status:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install clickup CLI
        run: |
          curl -sL https://github.com/triptechtravel/clickup-cli/releases/latest/download/clickup_linux_amd64.tar.gz | tar xz
          sudo mv clickup /usr/local/bin/
      - name: Authenticate
        run: echo "${{ "{{" }} secrets.CLICKUP_TOKEN {{ "}}" }}" | clickup auth login --with-token
      - name: Approved
        if: github.event.review.state == 'approved'
        run: clickup status set "awaiting testing"
      - name: Changes requested
        if: github.event.review.state == 'changes_requested'
        run: |
          clickup status set "in progress"
          clickup comment add "" "Changes requested by ${{ "{{" }} github.event.review.user.login {{ "}}" }}"
```

## Post CI results

Comments the CI result on the ClickUp task when a workflow completes.

```yaml
name: ClickUp CI Status
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ "{{" }} github.event.workflow_run.head_branch {{ "}}" }}
      - name: Install clickup CLI
        run: |
          curl -sL https://github.com/triptechtravel/clickup-cli/releases/latest/download/clickup_linux_amd64.tar.gz | tar xz
          sudo mv clickup /usr/local/bin/
      - name: Authenticate
        run: echo "${{ "{{" }} secrets.CLICKUP_TOKEN {{ "}}" }}" | clickup auth login --with-token
      - name: Post result
        run: |
          CONCLUSION="${{ "{{" }} github.event.workflow_run.conclusion {{ "}}" }}"
          BRANCH="${{ "{{" }} github.event.workflow_run.head_branch {{ "}}" }}"
          SHA="${{ "{{" }} github.event.workflow_run.head_sha {{ "}}" }}"
          clickup comment add "" "CI ${CONCLUSION}: ${BRANCH} (${SHA:0:7})"
```

See also: [Using with AI agents](/clickup-cli/ai-agents/) for integrating the CLI with AI coding tools.
