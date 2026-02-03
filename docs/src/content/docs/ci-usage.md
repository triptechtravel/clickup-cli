---
title: CI usage
description: Using the clickup CLI in CI/CD pipelines with JSON output and non-interactive auth.
---

# CI usage

The CLI works in non-interactive environments like GitHub Actions, GitLab CI, and other CI/CD pipelines.

## Authentication

Pass your ClickUp API token via stdin using `--with-token`:

```sh
echo "$CLICKUP_TOKEN" | clickup auth login --with-token
```

Store the token as a CI secret (e.g., GitHub Actions secret, GitLab CI variable). Do not pass it as a command-line argument.

## JSON output

All list and view commands support `--json` for machine-readable output:

```sh
clickup task view CU-abc123 --json
clickup sprint current --json
clickup inbox --json
```

### Filtering with `--jq`

Use `--jq` to extract specific fields inline:

```sh
# Get just the task name
clickup task view CU-abc123 --json --jq '.name'

# List sprint task names
clickup sprint current --json --jq '.[].name'

# Get task IDs in a specific status
clickup sprint current --json --jq '[.[] | select(.status == "in progress") | .id]'
```

## GitHub Actions example

```yaml
name: Update ClickUp
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  link-pr:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install clickup CLI
        run: |
          curl -sL https://github.com/triptechtravel/clickup-cli/releases/latest/download/clickup_linux_amd64.tar.gz | tar xz
          sudo mv clickup /usr/local/bin/

      - name: Authenticate
        run: echo "${{ secrets.CLICKUP_TOKEN }}" | clickup auth login --with-token

      - name: Link PR to ClickUp task
        run: clickup link pr ${{ github.event.pull_request.number }}
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (API failure, invalid input, etc.) |

Commands that produce no output (e.g., no tasks found) still exit with code 0.
