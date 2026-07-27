# Contributing to clickup-cli

Thank you for your interest in contributing! Full contributor documentation is at **[triptechtravel.github.io/clickup-cli/contributing](https://triptechtravel.github.io/clickup-cli/contributing/)**.

## Quick start

```bash
git clone https://github.com/triptechtravel/clickup-cli.git
cd clickup-cli

# Install codegen tool (one-time)
go install github.com/oapi-codegen/oapi-codegen-exp/experimental/cmd/oapi-codegen@latest

# Generate API clients from ClickUp OpenAPI specs
make api-gen

# Build and test
go build -o clickup ./cmd/clickup
go test ./...
go vet ./...
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
| `internal/apiv2/operations.gen.go` | Auto-generated wrappers using codegen types (used for tag/checklist/time operations) |
| `internal/apiv3/operations.gen.go` | Auto-generated V3 wrappers (chat, docs, audit) |
| `api/clickupv2/`, `api/clickupv3/` | Auto-generated request/response types from OpenAPI specs |
| `internal/api/` | HTTP client with auth, rate limiting, error handling |
| `pkg/cmd/*/` | CLI commands (cobra) |
| `pkg/cmdutil/` | Shared helpers (config, custom IDs, sprint resolution) |

### API codegen pipeline

Generated code is gitignored — run `make api-gen` after cloning.

```
make api-gen
├── Fetches V2 + V3 specs from developer.clickup.com
│   └── verified against SPEC_V2_SHA / SPEC_V3_SHA — see below
├── oapi-codegen → types (api/clickupv2/, api/clickupv3/)
├── gen-api -fix → resolves broken $refs via spec introspection (fixes.gen.go)
├── gen-api → typed wrapper functions (internal/apiv2/, internal/apiv3/)
└── gen-api -flags-out → cobra flag binding per request field (flags.gen.go)
```

**Specs are pinned by checksum.** Generated code is not committed, which makes
the spec the source of truth, so it cannot be fetched unpinned — two clones a
week apart would otherwise build different clients silently. A mismatch fails
the build with the expected and actual hashes.

When ClickUp publishes a new spec, re-pin deliberately:

```bash
make api-update          # re-pins the checksums
make api-clean api-gen   # regenerate
git diff                 # review what changed before committing
```

**Adding a request field costs no Go.** `gen-api` emits a `<Op>Flags` struct per
operation with a request body; `Register(cmd, skip...)` binds a flag per scalar
field and `Apply(req)` copies the ones the user set. A field added upstream
becomes a working flag after `make api-gen`. Commands skip any field they bind
themselves, where hand-written ergonomics beat a generic flag.

`internal/clickup.TestSpecDrift` fails the build when a hand-written type lacks
a field the spec has — the guard that exists because `archived` went missing
that way.

### When to use which layer

- **New commands** that need task/list/folder/space/team data: use `apiv2.*Local` wrappers in `local.go` — they decode into `internal/clickup` types
- **Operations already covered by generated wrappers** (tags, checklists, time entries): use `apiv2.*` from `operations.gen.go` with `clickupv2.*` request types
- **V3 API** (chat, docs): use `apiv3.*` from `operations.gen.go`
- **New API endpoints not yet generated**: add to `local.go` using the `do()` helper

### Adding a new command

1. Create `pkg/cmd/<group>/<command>.go`
2. Use `apiv2.*Local` for API calls (see `pkg/cmd/folder/list.go` for a clean example)
3. Register in `pkg/cmd/root/root.go`
4. Add tests in `pkg/cmd/<group>/<command>_test.go` (use `internal/testutil/`)
5. Run `go run ./cmd/gen-docs` to regenerate reference docs
6. Update `skills/clickup-cli/SKILL.md` with usage examples

## Submitting changes

1. Fork the repository and create your branch from `main`
2. Make your changes and add tests for new functionality
3. Run `make api-gen && go build ./... && go test ./... && go vet ./... && make lint`
4. Submit a pull request with a clear description

## Release process

Releases use [GoReleaser](https://goreleaser.com/) via GitHub Actions:

1. Tag the release: `git tag v0.x.0`
2. Push the tag: `git push origin v0.x.0`
3. GitHub Actions runs `make api-gen`, builds, and publishes binaries
4. Homebrew formula is auto-updated in `triptechtravel/homebrew-tap`

Install: `brew install triptechtravel/tap/clickup`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
