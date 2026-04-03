BINARY := clickup
MODULE := github.com/triptechtravel/clickup-cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
	-X $(MODULE)/internal/build.Version=$(VERSION) \
	-X $(MODULE)/internal/build.Commit=$(COMMIT) \
	-X $(MODULE)/internal/build.Date=$(DATE)

.PHONY: build install test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/clickup

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/clickup

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ dist/

.PHONY: docs
docs:
	go run ./cmd/gen-docs

.PHONY: snapshot
snapshot:
	goreleaser --snapshot --clean

# ── API spec + codegen ──────────────────────────────────────────────
SPEC_V2_URL := https://developer.clickup.com/openapi/clickup-api-v2-reference.json
SPEC_V3_URL := https://developer.clickup.com/openapi/ClickUp_PUBLIC_API_V3.yaml

api/specs/clickup-v2.json:
	@mkdir -p api/specs
	curl -sfL -o $@ $(SPEC_V2_URL)
	@echo "Downloaded V2 spec ($$(wc -c < $@ | tr -d ' ') bytes)"

api/specs/clickup-v3.yaml:
	@mkdir -p api/specs
	curl -sfL -o $@ $(SPEC_V3_URL)
	@echo "Downloaded V3 spec ($$(wc -c < $@ | tr -d ' ') bytes)"

.PHONY: api-spec
api-spec: api/specs/clickup-v2.json api/specs/clickup-v3.yaml

.PHONY: api-gen
api-gen: api-spec
	@echo "Generating types from specs..."
	cd api && go generate .
	@echo "Fixing self-referencing types (V2)..."
	go run ./cmd/gen-api -fix -spec api/specs/clickup-v2.json -fix-gen api/clickupv2/client.gen.go -fix-out api/clickupv2/fixes.gen.go -fix-pkg clickupv2
	@echo "Fixing V3 codegen issues..."
	perl -0pi -e 's/if s\.Parent == nil \{\n\t\tv := "null"\n\t\ts\.Parent = &v\n\t\}//' api/clickupv3/client.gen.go
	@echo "Generating API wrappers..."
	go run ./cmd/gen-api -spec api/specs/clickup-v2.json -pkg apiv2 -types-pkg clickupv2 -out internal/apiv2/operations.gen.go
	go run ./cmd/gen-api -spec api/specs/clickup-v3.yaml -pkg apiv3 -types-pkg clickupv3 -out internal/apiv3/operations.gen.go
	@echo "Done: V2 + V3 types, fixes, and wrappers generated."

.PHONY: api-clean
api-clean:
	rm -rf api/specs/ api/clickupv2/*.gen.go api/clickupv3/*.gen.go

# ── Skills ──────────────────────────────────────────────────────────
.PHONY: install-skill
install-skill:
	@mkdir -p $(HOME)/.claude/skills
	@ln -sfn $(CURDIR)/skills/clickup-cli $(HOME)/.claude/skills/clickup-cli
	@echo "Linked Claude Code skill: clickup-cli → ~/.claude/skills/clickup-cli"
	@echo ""
	@echo "Or install via plugin marketplace (no clone required):"
	@echo "  /plugin marketplace add triptechtravel/clickup-cli"
	@echo "  /plugin install clickup-cli@clickup-cli"
