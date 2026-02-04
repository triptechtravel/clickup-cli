---
title: "clickup task search"
description: "Auto-generated reference for clickup task search"
---

## clickup task search

Search tasks by name

### Synopsis

Search ClickUp tasks across the workspace by name.

Returns tasks whose names match the search query using substring and fuzzy
matching. Exact substring matches are shown first, followed by fuzzy matches
sorted by relevance.

Use --space and --folder to narrow the search scope for faster results.
Use --comments to also search through task comments (slower).

In interactive mode (TTY), if many results are found you will be asked
whether to refine the search. Use --pick to interactively select a single
task and print only its ID.

When no exact match is found, the search automatically tries individual
words from the query and shows potentially related tasks.

If search returns no results, use 'clickup task recent' to see your
recently updated tasks and discover which folders/lists to search in.

```
clickup task search <query> [flags]
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

  # Interactively pick a task (prints selected task ID)
  clickup task search geozone --pick

  # If search returns no results, find your active folders first
  clickup task recent
  clickup task search geozone --folder "Engineering Sprint"

  # JSON output
  clickup task search geozone --json
```

### Options

```
      --comments          Also search through task comments (slower)
      --exact             Only show exact substring matches (no fuzzy results)
      --folder string     Limit search to a specific folder (name, substring match)
  -h, --help              help for search
      --jq string         Filter JSON output using a jq expression
      --json              Output JSON
      --pick              Interactively select a task and print its ID
      --space string      Limit search to a specific space (name or ID)
      --template string   Format JSON output using a Go template
```

### SEE ALSO

* [clickup task](/clickup-cli/reference/clickup_task/)	 - Manage ClickUp tasks

