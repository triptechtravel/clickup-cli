---
title: "clickup task search"
description: "Auto-generated reference for clickup task search"
---

Search tasks by name and description

### Synopsis

Search ClickUp tasks across the workspace by name and description.

Returns tasks whose names or descriptions match the search query. Matching
priority: name substring > name fuzzy > description substring.

ClickUp has no server-side text search, so matching happens locally. To keep
that from meaning "paginate the workspace on every search", the CLI keeps an
index of the workspace under ~/.cache/clickup (override with CLICKUP_CACHE_DIR)
and tops it up with only what changed since the last run. The first search
builds the index and is slow; later ones read every task in the workspace for
about one request. The index is rebuilt weekly so that deleted and archived
tasks fall out of it.

Use --no-cache to skip the index and query the API directly. That path reads
only the most recently updated tasks and says so when it runs out of budget.
--comments and --assignee also bypass the index.

Use --space and --folder to search a specific part of the tree instead; this
reaches tasks of any age, at the cost of walking every list.

In interactive mode (TTY), if many results are found you will be asked
whether to refine the search. Use --pick to interactively select a single
task and print only its ID.

When no exact match is found, the search automatically tries individual
words from the query and shows potentially related tasks.

If search returns no results, use 'clickup task recent' to see your
recently updated tasks and discover which folders/lists to search in.

```
clickup task search [query] [flags]
```

### Examples

```
  # Search for tasks mentioning "payload"
  clickup task search payload

  # Search within a specific space
  clickup task search geozone --space Development

  # Search within a specific folder
  clickup task search nextjs --folder "Engineering sprint"

  # Also search through task comments
  clickup task search "migration issue" --comments

  # Filter by assignee
  clickup task search --assignee me
  clickup task search "bug" --assignee "Isaac"
  clickup task search --assignee 54695018

  # Interactively pick a task (prints selected task ID)
  clickup task search geozone --pick

  # If search returns no results, find your active folders first
  clickup task recent
  clickup task search geozone --folder "Engineering Sprint"

  # Include subtasks in results
  clickup task search "Phase 1" --include-subtasks

  # JSON output
  clickup task search geozone --json
```

### Options

```
      --assignee string    Filter by assignee (name, username, numeric ID, or "me")
      --comments           Also search through task comments (slower)
      --exact              Only show exact substring matches (no fuzzy results)
      --folder string      Limit search to a specific folder (name, substring match)
  -h, --help               help for search
      --include-subtasks   Include subtasks in search results
      --jq string          Filter JSON output using a jq expression
      --json               Output JSON
      --no-cache           Bypass the local task index and query the API directly
      --pick               Interactively select a task and print its ID
  -r, --raw                Output raw strings instead of JSON-encoded (use with --jq)
      --space string       Limit search to a specific space (name or ID)
      --template string    Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

