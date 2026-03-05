# Contributing to clickup-cli

Thank you for your interest in contributing! Full contributor documentation is at **[triptechtravel.github.io/clickup-cli/contributing](https://triptechtravel.github.io/clickup-cli/contributing/)**.

## Quick start

```bash
git clone https://github.com/triptechtravel/clickup-cli.git
cd clickup-cli
go build -o clickup ./cmd/clickup
go test ./...
go vet ./...
```

Requires Go 1.25 or later.

## Submitting changes

1. Fork the repository and create your branch from `main`
2. Make your changes and add tests for new functionality
3. Run `go test ./...` and `go vet ./...`
4. Submit a pull request with a clear description

## Release process

Releases use [GoReleaser](https://goreleaser.com/) via GitHub Actions:

1. Tag the release: `git tag v0.x.0`
2. Push the tag: `git push origin v0.x.0`
3. GitHub Actions builds and publishes binaries
4. Update the Homebrew formula in `triptechtravel/homebrew-tap`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
