# Contributing to clickup-cli

Thank you for your interest in contributing! This document provides guidelines and information for contributors.

## How to Contribute

### Reporting Bugs

Before creating a bug report, please check existing issues to avoid duplicates. When creating a bug report, include:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected vs actual behavior
- Your environment (OS, Go version, clickup-cli version)
- Any relevant output or error messages

### Suggesting Features

Feature requests are welcome! Please include:

- A clear description of the feature
- The problem it solves or use case it enables
- Any implementation ideas you have

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Install Go 1.22+**
3. **Make your changes** following the code style guidelines below
4. **Add tests** for any new functionality
5. **Run the test suite**: `go test ./...`
6. **Run vet**: `go vet ./...`
7. **Update documentation** if needed
8. **Submit your pull request**

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/clickup-cli.git
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

## Project Structure

```
cmd/clickup/          Entry point
internal/             Internal packages (not importable)
  api/                HTTP client, rate limiting
  app/                Bootstrap, root command wiring
  auth/               Keyring storage, OAuth, token validation
  build/              Version info (ldflags)
  config/             YAML config loading/saving
  git/                Branch detection, task ID extraction
  iostreams/          TTY detection, color support
  tableprinter/       Table rendering
  prompter/           Interactive prompts
  browser/            Cross-platform browser opening
  text/               String helpers
pkg/cmd/              Command implementations
  auth/               login, logout, status
  task/               view, list, create, edit
  comment/            add, list, edit
  status/             set, list
  link/               pr, branch, commit
  sprint/             list, current
  space/              list, select
  inbox/              inbox
  completion/         shell completions
  version/            version
```

## Code Style Guidelines

- **Go conventions**: Follow standard Go formatting (`gofmt`) and naming
- **Error handling**: Wrap errors with context using `fmt.Errorf("...: %w", err)`
- **Naming**: Use camelCase for unexported, PascalCase for exported
- **Testing**: Use table-driven tests where appropriate
- **No unnecessary dependencies**: Prefer the standard library unless a dependency adds clear value

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./pkg/cmd/sprint/...

# Run a specific test
go test -run TestClassifySprint ./pkg/cmd/sprint/
```

## Commit Messages

Write clear commit messages that describe the change:

- `Add support for custom task ID prefixes`
- `Fix fuzzy status matching for multi-word statuses`
- `Update README with Homebrew install instructions`

## Release Process

Releases are managed by maintainers using GoReleaser:

1. Tag the release: `git tag v0.2.0`
2. Push the tag: `git push origin v0.2.0`
3. GitHub Actions builds and publishes binaries
4. Update the Homebrew formula in `triptechtravel/homebrew-tap`

## Questions?

Feel free to open an issue for any questions about contributing.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
