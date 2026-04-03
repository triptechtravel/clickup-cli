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

## API codegen pipeline

The CLI uses auto-generated Go types and wrapper functions from the official ClickUp OpenAPI specs. Generated code is gitignored — you must run `make api-gen` after cloning.

```
make api-gen
├── Fetches V2 + V3 specs from developer.clickup.com
├── oapi-codegen → types (api/clickupv2/, api/clickupv3/)
├── gen-api -fix → resolves broken $refs via spec introspection (fixes.gen.go)
└── gen-api → typed wrapper functions (internal/apiv2/, internal/apiv3/)
```

- `api/clickupv2/` — auto-gen types for all 137 V2 operations
- `api/clickupv3/` — auto-gen types for 18 V3 operations (chat, docs, audit logs)
- `internal/apiv2/` — typed wrappers using auto-gen types + `api.Client` transport
- `internal/apiv3/` — same for V3
- `cmd/gen-api/` — the code generator (reads spec, emits Go)

go-clickup (`raksul/go-clickup`) is kept for operations it handles correctly. The auto-gen layer fills gaps documented in `api/GO_CLICKUP_GAPS.md`.

## Submitting changes

1. Fork the repository and create your branch from `main`
2. Make your changes and add tests for new functionality
3. Run `make api-gen && go test ./... && go vet ./...`
4. Submit a pull request with a clear description

## Release process

Releases use [GoReleaser](https://goreleaser.com/) via GitHub Actions:

1. Tag the release: `git tag v0.x.0`
2. Push the tag: `git push origin v0.x.0`
3. GitHub Actions runs `make api-gen`, builds, and publishes binaries
4. Homebrew formula is auto-updated in `triptechtravel/homebrew-tap`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
