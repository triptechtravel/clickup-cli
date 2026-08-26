---
title: "clickup task search"
description: "Auto-generated reference for clickup task search"
---

Search tasks by name and description

### Synopsis

Search ClickUp tasks across the workspace by name and description.

Returns tasks whose names or descriptions match the search query. Matching
priority: name substring > name fuzzy > description substring.

ClickUp has no server-side text search, so matching happens locally. To avoid
paginating the workspace on every search, the CLI keeps an index under
~/.cache/clickup (override with CLICKUP_CACHE_DIR) and tops it up with only
what changed. The first search builds it and is slow — around 25s for 4,000
tasks; a workspace too large to build in one pass is continued by the next
search rather than left incomplete. Later searches read every indexed task for
about one request.

Limits, all of which are disclosed on stderr when they bite:
  - at most 200 rows are returned
  - descriptions are indexed to their first 4096 bytes
  - with --comments, comments are checked on at most 100 tasks
  - the index is reconciled weekly, so a task deleted upstream can linger
    until then; --refresh rebuilds it immediately

Use --no-cache to skip the index and sweep the API directly; that path reads
only the most recently updated tasks. --comments and --assignee bypass the
index too. --space and --folder search a specific part of the tree instead,
reaching tasks of any age at the cost of walking every list.

The index holds task names and descriptions in plaintext. 'clickup auth
logout' deletes it.

--json emits `parent` (empty for top-level tasks) and `date_updated` (epoch
milliseconds, as a string) alongside the task fields.

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
      --no-cache           Bypass the local task index and sweep the API directly (no effect with --space/--folder, which never use the index)
      --pick               Interactively select a task and print its ID
  -r, --raw                Output raw strings instead of JSON-encoded (use with --jq)
      --refresh            Rebuild the local task index from scratch before searching
      --space string       Limit search to a specific space (name or ID)
      --template string    Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

