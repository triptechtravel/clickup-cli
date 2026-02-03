---
title: Contributing
description: Development setup, project structure, and contribution guidelines.
---

# Contributing

Thank you for your interest in contributing to clickup-cli! This page covers development setup, project structure, and contribution guidelines.

## Development setup

```sh
# Clone the repository
git clone https://github.com/triptechtravel/clickup-cli.git
cd clickup-cli

# Build
go build -o clickup ./cmd/clickup

# Run tests
go test ./...

# Run vet
go vet ./...

# Install locally
go install ./cmd/clickup
```

Requires Go 1.22 or later.

## Project structure

```
cmd/clickup/          Entry point
internal/             Internal packages (not importable)
  api/                HTTP client, rate limiting, error handling
  app/                Bootstrap and root command wiring
  auth/               Keyring storage, OAuth flow, token validation
  build/              Version info (injected via ldflags)
  config/             YAML config loading and saving
  git/                Branch detection, task ID extraction
  iostreams/          TTY detection, color support, stream abstraction
  tableprinter/       TTY-aware table rendering
  prompter/           Interactive prompts (survey-based)
  browser/            Cross-platform browser opening
  text/               String helpers (truncate, pluralize, relative time)
pkg/
  cmdutil/            Dependency injection, JSON flags, auth middleware
  cmd/                Command implementations
    auth/             login, logout, status
    task/             view, list, create, edit
    comment/          add, list, edit
    status/           set, list
    link/             pr, branch, commit
    sprint/           list, current
    space/            list, select
    inbox/            inbox (@mentions)
    completion/       shell completions
    version/          version
```

## Running tests

```sh
# Run all tests
go test ./...

# Verbose output
go test -v ./...

# Specific package
go test ./pkg/cmd/sprint/...

# Specific test
go test -run TestClassifySprint ./pkg/cmd/sprint/
```

## Code style

- Follow standard Go formatting (`gofmt`)
- Wrap errors with context: `fmt.Errorf("failed to fetch tasks: %w", err)`
- Use table-driven tests where appropriate
- Prefer the standard library unless a dependency adds clear value

## Submitting changes

1. Fork the repository and create your branch from `main`
2. Make your changes and add tests for new functionality
3. Run `go test ./...` and `go vet ./...`
4. Submit a pull request with a clear description

## Release process

Releases use [GoReleaser](https://goreleaser.com/) via GitHub Actions:

1. Tag the release: `git tag v0.2.0`
2. Push the tag: `git push origin v0.2.0`
3. GitHub Actions builds binaries for all platforms
4. Update the Homebrew formula in `triptechtravel/homebrew-tap`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](https://github.com/triptechtravel/clickup-cli/blob/main/LICENSE).
