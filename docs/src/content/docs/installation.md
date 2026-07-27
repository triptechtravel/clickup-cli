---
title: Installation
description: Install the clickup CLI via the install script, Go, Homebrew, or binary release.
---

There are four ways to install the `clickup` CLI.

## Install script

On Linux and macOS, the quickest option:

```sh
curl -fsSL https://raw.githubusercontent.com/triptechtravel/clickup-cli/main/scripts/install.sh | sh
```

This detects your OS and architecture, downloads the matching release, verifies its SHA-256 checksum, and installs the binary to `/usr/local/bin` (falling back to `~/.local/bin` if that isn't writable).

To review the script before running it — always a good idea for `curl | sh` — download it first:

```sh
curl -fsSL -o install.sh https://raw.githubusercontent.com/triptechtravel/clickup-cli/main/scripts/install.sh
less install.sh
sh install.sh
```

The script honours a few environment variables:

| Variable | Purpose |
| --- | --- |
| `CLICKUP_VERSION` | Install a specific tag (e.g. `v0.35.1`) instead of the latest release. |
| `CLICKUP_INSTALL_DIR` | Install to a specific directory. |
| `CLICKUP_NO_SUDO` | Set to `1` to never escalate; installs to `~/.local/bin` instead. |

```sh
# Pin a version and install somewhere on your own PATH
CLICKUP_VERSION=v0.35.1 CLICKUP_INSTALL_DIR="$HOME/bin" sh install.sh
```

Windows isn't supported by the script — use a [binary release](#binary-releases).

### Supported platforms

The script installs prebuilt binaries for Linux and macOS on `amd64` and `arm64`. On any other architecture it exits with a message telling you what's available.

Because the binaries are statically linked with no libc dependency, a few environments work without any special handling:

| Environment | Notes |
| --- | --- |
| **WSL2** | A normal Linux install — use the script as-is. |
| **Docker and CI containers** | Works on Alpine and other musl-based images, not just glibc ones. |
| **Raspberry Pi** | Works on 64-bit Raspberry Pi OS (`arm64`). The older 32-bit builds aren't supported. |
| **Headless servers** | No desktop keyring required — see below. |

On a machine with no OS keyring (a container, or a headless server without a D-Bus secret service), `clickup auth login` falls back to storing your token as plain text in the config directory and prints a warning. That's expected. In CI, pipe the token in from a secret rather than logging in interactively — see [CI usage](/clickup-cli/ci-usage/):

```sh
echo "$CLICKUP_TOKEN" | clickup auth login --with-token
```

Set `CLICKUP_CONFIG_DIR` to control where that config lives, which is useful for keeping it on a writable volume in a container.

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
