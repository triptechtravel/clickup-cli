---
title: Contributing
description: Development setup, project structure, and contribution guidelines.
---

Thank you for your interest in contributing to clickup-cli! This page covers development setup, project structure, and contribution guidelines.

## Development setup

```sh
# Clone the repository
git clone https://github.com/triptechtravel/clickup-cli.git
cd clickup-cli

# Install codegen tool (one-time)
go install github.com/oapi-codegen/oapi-codegen-exp/experimental/cmd/oapi-codegen@latest

# Generate API clients from ClickUp OpenAPI specs
make api-gen

# Build
go build -o clickup ./cmd/clickup

# Run tests
go test ./...

# Run vet
go vet ./...

# Install locally
go install ./cmd/clickup

# Regenerate CLI reference docs (Starlight markdown)
make docs
```

Requires Go 1.25 or later.

## API codegen pipeline

The CLI uses auto-generated Go types and wrapper functions from the official ClickUp OpenAPI specifications. Generated code is gitignored — you must run `make api-gen` after cloning.

```
make api-gen
├── Fetches V2 + V3 specs from developer.clickup.com
├── oapi-codegen → types (api/clickupv2/, api/clickupv3/)
├── gen-api -fix → resolves broken $refs via spec introspection (fixes.gen.go)
└── gen-api → typed wrapper functions (internal/apiv2/, internal/apiv3/)
```

The codegen tool (`cmd/gen-api/`) reads the OpenAPI spec and emits:
- **Types** — request/response structs matching the spec exactly
- **Fixes** — introspects the spec to resolve `$ref` chains that the type generator can't handle, emitting proper struct definitions
- **Wrappers** — one typed Go function per API operation, routed through `api.Client` (auth + rate limiting)

The go-clickup library (`raksul/go-clickup`) is kept for operations it handles correctly. Auto-gen wrappers fill the gaps (documented in `api/GO_CLICKUP_GAPS.md`).

To regenerate after spec changes or codegen updates:

```sh
make api-clean   # remove all generated files
make api-gen     # re-fetch specs + regenerate everything
```

## Project structure

```
cmd/clickup/          Entry point
cmd/gen-docs/         Auto-generates CLI reference markdown (make docs)
cmd/gen-api/          API codegen tool (reads OpenAPI spec, emits Go)
api/
  specs/              OpenAPI specs (gitignored, fetched on demand)
  clickupv2/          Auto-generated V2 types (137 operations)
  clickupv3/          Auto-generated V3 types (18 operations)
internal/             Internal packages (not importable)
  api/                HTTP client, rate limiting, error handling
  apiv2/              Auto-generated V2 wrapper functions
  apiv3/              Auto-generated V3 wrapper functions
  testutil/           Shared test infrastructure (mock server, factory)
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
  cmdutil/            Dependency injection, JSON flags, auth middleware,
                      recent tasks helper
  cmd/                Command implementations
    auth/             login, logout, status
    task/             view, list, create, edit, search, recent, activity,
                      time, dependency, checklist, custom fields
    comment/          add, list, edit, delete, reply
    status/           set, list, add
    link/             pr, branch, commit, sync, description upsert,
                      custom field linking
    field/            list (custom field discovery)
    sprint/           list, current
    space/            list, select
    tag/              list, create
    member/           list
    inbox/            inbox (@mentions)
    completion/       shell completions
    version/          version
```

## Running tests

```sh
# Run all tests
go test ./...

# With race detector
go test -race ./...

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
- For new API endpoints, prefer auto-gen wrappers (`internal/apiv2/`) over raw HTTP calls

## Submitting changes

1. Fork the repository and create your branch from `main`
2. Make your changes and add tests for new functionality
3. Run `make api-gen && go test ./... && go vet ./...`
4. Submit a pull request with a clear description

## Release process

Releases use [GoReleaser](https://goreleaser.com/) via GitHub Actions:

1. Tag the release: `git tag v0.x.0`
2. Push the tag: `git push origin v0.x.0`
3. GitHub Actions runs `make api-gen`, builds binaries for all platforms
4. Homebrew formula is auto-updated in `triptechtravel/homebrew-tap`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](https://github.com/triptechtravel/clickup-cli/blob/main/LICENSE).
