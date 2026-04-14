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

## Architecture

All ClickUp API calls go through a single code path:

```
CLI command (pkg/cmd/*)
  → apiv2/local.go wrappers (typed, decode into internal/clickup types)
  → apiv2.do() helper (JSON marshal, HTTP transport, error handling)
  → api.Client.DoRequest() (auth header, rate limiting, 429 retry)
```

### Key packages

| Package | Purpose |
|---------|---------|
| `internal/clickup/` | Local type definitions (Task, List, Folder, etc.) with stable JSON tags |
| `internal/apiv2/local.go` | Typed wrappers that decode API responses into local types |
| `internal/apiv2/operations.gen.go` | Auto-generated wrappers using codegen types (tags, checklists, time entries) |
| `internal/apiv3/operations.gen.go` | Auto-generated V3 wrappers (chat, docs, audit) |
| `api/clickupv2/`, `api/clickupv3/` | Auto-generated request/response types from OpenAPI specs |
| `internal/api/` | HTTP client with auth, rate limiting, error handling |
| `pkg/cmd/*/` | CLI commands (cobra) |
| `pkg/cmdutil/` | Shared helpers (config, custom IDs, sprint resolution) |

### When to use which layer

- **New commands** that need task/list/folder/space/team data: use `apiv2.*Local` wrappers in `local.go`
- **Operations already covered by generated wrappers** (tags, checklists, time entries): use `apiv2.*` from `operations.gen.go` with `clickupv2.*` request types
- **V3 API** (chat, docs): use `apiv3.*` from `operations.gen.go`
- **New API endpoints not yet generated**: add to `local.go` using the `do()` helper

## API codegen pipeline

Generated code is gitignored — run `make api-gen` after cloning.

```
make api-gen
├── Fetches V2 + V3 specs from developer.clickup.com
├── oapi-codegen → types (api/clickupv2/, api/clickupv3/)
├── gen-api -fix → resolves broken $refs via spec introspection (fixes.gen.go)
└── gen-api → typed wrapper functions (internal/apiv2/, internal/apiv3/)
```

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
  clickupv2/          Auto-generated V2 types (135 operations)
  clickupv3/          Auto-generated V3 types (31 operations)
internal/             Internal packages (not importable)
  clickup/            Local type definitions (Task, List, Date, CustomField, etc.)
  api/                HTTP client, rate limiting, error handling
  apiv2/              V2 API wrappers (local.go + auto-generated operations)
  apiv3/              V3 API wrappers (auto-generated)
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
                      time, dependency, checklist, custom fields, delete
    comment/          add, list, edit, delete, reply
    status/           set, list, add
    link/             pr, branch, commit, sync, description upsert,
                      custom field linking
    field/            list (custom field discovery)
    sprint/           list, current
    space/            list, select
    folder/           list, select
    list/             list, select
    chat/             send
    tag/              list, create
    member/           list
    inbox/            inbox (@mentions)
    attachment/       add
    doc/              list, view, create, page list/view/create/edit
    completion/       shell completions
    version/          version
```

## Adding a new command

1. Create `pkg/cmd/<group>/<command>.go`
2. Use `apiv2.*Local` for API calls (see `pkg/cmd/folder/list.go` for a clean example)
3. Register in `pkg/cmd/root/root.go`
4. Add tests in `pkg/cmd/<group>/<command>_test.go` (use `internal/testutil/`)
5. Run `go run ./cmd/gen-docs` to regenerate reference docs
6. Update `skills/clickup-cli/SKILL.md` with usage examples

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
- For new API endpoints, add typed wrappers to `internal/apiv2/local.go`
- Use `internal/clickup` types for response decoding (stable JSON tags)

## Submitting changes

1. Fork the repository and create your branch from `main`
2. Make your changes and add tests for new functionality
3. Run `make api-gen && go build ./... && go test ./... && go vet ./...`
4. Submit a pull request with a clear description

## Release process

Releases use [GoReleaser](https://goreleaser.com/) via GitHub Actions:

1. Tag the release: `git tag v0.x.0`
2. Push the tag: `git push origin v0.x.0`
3. GitHub Actions runs `make api-gen`, builds binaries for all platforms
4. Homebrew formula is auto-updated in `triptechtravel/homebrew-tap`

Install: `brew install triptechtravel/tap/clickup`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](https://github.com/triptechtravel/clickup-cli/blob/main/LICENSE).
