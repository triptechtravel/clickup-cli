---
title: "clickup auth login"
description: "Auto-generated reference for clickup auth login"
---

## clickup auth login

Authenticate with ClickUp

### Synopsis

Authenticate with a ClickUp account.

By default, this command prompts for a personal API token.
Get your token from ClickUp > Settings > ClickUp API > API tokens.

In non-interactive environments (CI), pipe a token via stdin:
  echo "pk_12345" | clickup auth login --with-token

To use OAuth instead (requires a registered OAuth app):
  clickup auth login --oauth

```
clickup auth login [flags]
```

### Examples

```
  # Interactive token entry (default)
  clickup auth login

  # Pipe token for CI
  echo "pk_12345" | clickup auth login --with-token

  # Use OAuth browser flow
  clickup auth login --oauth
```

### Options

```
  -h, --help         help for login
      --oauth        Use OAuth browser flow (requires registered OAuth app)
      --with-token   Read token from standard input (for CI)
```

### SEE ALSO

* [clickup auth](/clickup-cli/reference/clickup_auth/)	 - Authenticate with ClickUp

