---
title: Installation
description: Install the clickup CLI via Go, Homebrew, or binary release.
---

# Installation

There are three ways to install the `clickup` CLI.

## Go

If you have Go 1.25 or later installed, use `go install`:

```sh
go install github.com/triptechtravel/clickup-cli/cmd/clickup@latest
```

This places the binary in your `$GOBIN` directory (typically `$HOME/go/bin`). Make sure that directory is in your `PATH`.

## Homebrew

On macOS and Linux, install via Homebrew:

```sh
brew install triptechtravel/tap/clickup
```

To upgrade later:

```sh
brew upgrade triptechtravel/tap/clickup
```

## Binary releases

Download a prebuilt binary for your platform from the [GitHub releases page](https://github.com/triptechtravel/clickup-cli/releases).

1. Download the archive for your OS and architecture.
2. Extract it.
3. Move the `clickup` binary to a directory in your `PATH` (for example, `/usr/local/bin`).

```sh
# Example for macOS arm64
tar xzf clickup_darwin_arm64.tar.gz
sudo mv clickup /usr/local/bin/
```

## Verify the installation

After installing, confirm the CLI is available:

```sh
clickup version
```

This prints the version, commit SHA, and build date.

## Shell completions

Generate completion scripts for your shell to enable tab completion of commands and flags.

```sh
# Bash
source <(clickup completion bash)

# Zsh
source <(clickup completion zsh)
# Or install permanently:
clickup completion zsh > "${fpath[1]}/_clickup"

# Fish
clickup completion fish | source

# PowerShell
clickup completion powershell | Out-String | Invoke-Expression
```

## Dependencies

The `link pr` command requires the [GitHub CLI](https://cli.github.com/) (`gh`) to be installed and authenticated. Install it separately if you plan to link pull requests to ClickUp tasks.
