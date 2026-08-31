BINARY := clickup
MODULE := github.com/triptechtravel/clickup-cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
	-X $(MODULE)/internal/build.Version=$(VERSION) \
	-X $(MODULE)/internal/build.Commit=$(COMMIT) \
	-X $(MODULE)/internal/build.Date=$(DATE)

.PHONY: build install test lint clean smoke test-install ensure-gen

ensure-gen:
	@[ -f api/clickupv2/client.gen.go ] && [ -f api/clickupv3/client.gen.go ] || $(MAKE) api-gen

build: ensure-gen
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/clickup

install: ensure-gen
	go install -ldflags "$(LDFLAGS)" ./cmd/clickup

test: ensure-gen
	go test ./...

# Round-trip the typed-wrapper code paths against a real ClickUp workspace.
# Catches OpenAPI spec drift unit tests can't see. Requires `clickup auth login`.
# Override BIN to test a local build: `BIN=./bin/clickup make smoke`.
smoke:
	@./scripts/smoke.sh

# Exercise scripts/install.sh across shells, downloaders, and privilege
# levels in Docker. Filter with `./scripts/test-install.sh -k alpine`.
test-install:
	@./scripts/test-install.sh

lint: fmt-check
	golangci-lint run ./...

# Formatting is checked, not assumed. Every non-generated file was unformatted
# at some point because nothing enforced it.
.PHONY: fmt-check
fmt-check:
	@files=$$(gofmt -l . 2>/dev/null | grep -v '\.gen\.go' | grep -v '^\.claude/'); \
	if [ -n "$$files" ]; then \
		echo "These files are not gofmt-formatted:"; echo "$$files"; \
		echo ""; echo "Run: make fmt"; \
		exit 1; \
	fi

# Everything CI runs, in the same order, from regenerated code.
#
# Added after a green local run shipped a broken build: the generated client is
# gitignored, so a stale local copy can pass tests that CI — which regenerates
# from source — fails. Regenerating first is the only way "works locally" means
# what it says.
.PHONY: check
check:
	$(MAKE) api-clean api-gen
	go build ./...
	go test ./...
	go vet ./...
	$(MAKE) fmt-check
	golangci-lint run ./... || go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run ./...
	$(MAKE) docs
	@git diff --exit-code docs/ || { echo "Docs are out of date; commit the regenerated docs."; exit 1; }
	@echo "All CI checks pass."

.PHONY: fmt
fmt:
	@gofmt -w $$(git ls-files '*.go' | grep -v '\.gen\.go')
	@echo "Formatted."

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

# Generated code is deliberately not committed, which makes the spec the source
# of truth — so the spec is pinned. Without this, two clean clones a week apart
# can silently produce different clients. Bump deliberately with `make api-update`.
SPEC_V2_SHA := a0a72ec97ddb4e4859b9ed89b997bb784ba5828412ff35119f41e87103069662
SPEC_V3_SHA := 167e0b99e0c2218312d1318fff613f180dccdfcb8decb724ce08559aa04329f2

# The generator is pinned for the same reason the specs are: `make check`
# regenerates and compares, which proves nothing if the tool doing the
# generating floats. CI used to install this at @latest.
OAPI_CODEGEN := github.com/oapi-codegen/oapi-codegen-exp/experimental/cmd/oapi-codegen
OAPI_CODEGEN_VERSION := v0.0.0-20260414001447-ff920b08e315

.PHONY: tools
tools:
	go install $(OAPI_CODEGEN)@$(OAPI_CODEGEN_VERSION)

# verify_spec <file> <expected-sha> <name>
define verify_spec
	@actual=$$(shasum -a 256 $(1) | cut -d' ' -f1); \
	if [ "$$actual" != "$(2)" ]; then \
		echo ""; \
		echo "ERROR: $(3) spec checksum mismatch."; \
		echo "  expected: $(2)"; \
		echo "  actual:   $$actual"; \
		echo ""; \
		echo "Upstream published a new spec. Review the change, then re-pin:"; \
		echo "    make api-update"; \
		echo ""; \
		rm -f $(1); \
		exit 1; \
	fi
endef

# Depends on the patch: editing patch-v2-spec.jq has to re-derive the spec, or a
# tree that already has one silently keeps generating from the old rules — and
# `make api-gen` looks up to date, so only `make api-clean` recovers.
api/specs/clickup-v2.json: api/specs/patch-v2-spec.jq
	@mkdir -p api/specs
	curl -sfL -o $@.raw $(SPEC_V2_URL)
	$(call verify_spec,$@.raw,$(SPEC_V2_SHA),V2)
	@echo "Patching V2 spec (fixing time_spent, assignees, tags types)..."
	jq -f api/specs/patch-v2-spec.jq $@.raw > $@
	rm -f $@.raw
	@echo "Downloaded + patched V2 spec ($$(wc -c < $@ | tr -d ' ') bytes)"

api/specs/clickup-v3.yaml:
	@mkdir -p api/specs
	curl -sfL -o $@ $(SPEC_V3_URL)
	$(call verify_spec,$@,$(SPEC_V3_SHA),V3)
	@echo "Downloaded V3 spec ($$(wc -c < $@ | tr -d ' ') bytes)"

.PHONY: api-spec
api-spec: api/specs/clickup-v2.json api/specs/clickup-v3.yaml

.PHONY: api-gen
api-gen: api-spec
	@command -v oapi-codegen > /dev/null || { \
		echo "oapi-codegen not on PATH. Run: make tools"; exit 1; }
	@echo "Generating types from specs..."
	cd api && go generate .
	@echo "Fixing self-referencing types (V2)..."
	go run ./cmd/gen-api -fix -spec api/specs/clickup-v2.json -fix-gen api/clickupv2/client.gen.go -fix-out api/clickupv2/fixes.gen.go -fix-pkg clickupv2
	@echo "Fixing V3 codegen issues..."
	perl -0pi -e 's/if s\.Parent == nil \{\n\t\tv := "null"\n\t\ts\.Parent = &v\n\t\}//' api/clickupv3/client.gen.go
	@echo "Generating API wrappers..."
	go run ./cmd/gen-api -spec api/specs/clickup-v2.json -pkg apiv2 -types-pkg clickupv2 -out internal/apiv2/operations.gen.go -flags-out internal/apiv2/flags.gen.go
	go run ./cmd/gen-api -spec api/specs/clickup-v3.yaml -pkg apiv3 -types-pkg clickupv3 -out internal/apiv3/operations.gen.go
	@echo "Done: V2 + V3 types, fixes, and wrappers generated."

.PHONY: api-update
api-update:
	@echo "Fetching current specs to re-pin..."
	@mkdir -p api/specs
	@curl -sfL -o /tmp/clickup-v2-repin.json $(SPEC_V2_URL)
	@curl -sfL -o /tmp/clickup-v3-repin.yaml $(SPEC_V3_URL)
	@v2=$$(shasum -a 256 /tmp/clickup-v2-repin.json | cut -d' ' -f1); \
	 v3=$$(shasum -a 256 /tmp/clickup-v3-repin.yaml | cut -d' ' -f1); \
	 if [ "$$v2" = "$(SPEC_V2_SHA)" ] && [ "$$v3" = "$(SPEC_V3_SHA)" ]; then \
		echo "Specs unchanged; nothing to re-pin."; \
	 else \
		sed -i.bak -e "s/^SPEC_V2_SHA := .*/SPEC_V2_SHA := $$v2/" -e "s/^SPEC_V3_SHA := .*/SPEC_V3_SHA := $$v3/" Makefile && rm -f Makefile.bak; \
		echo "Re-pinned:"; echo "  V2 $$v2"; echo "  V3 $$v3"; \
		echo "Now run 'make api-clean api-gen' and review the generated diff."; \
	 fi
	@rm -f /tmp/clickup-v2-repin.json /tmp/clickup-v3-repin.yaml

.PHONY: api-clean
api-clean:
	rm -f api/specs/clickup-v2.json api/specs/clickup-v2.json.raw api/specs/clickup-v3.yaml
	rm -f api/clickupv2/*.gen.go api/clickupv3/*.gen.go

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
