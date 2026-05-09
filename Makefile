# kasapi-cli build helpers. Each target is a single shell line so it
# behaves the same under bash and fish.

GO       ?= go
BIN      ?= kasapi-cli
PKG      := ./cmd/kasapi-cli
CLI_DOCS := docs/cli

.PHONY: help build test lint vet fmt docs release-snapshot release-check clean

help:
	@echo "Targets:"
	@echo "  build             — go build ./cmd/kasapi-cli"
	@echo "  test              — go test ./..."
	@echo "  lint              — golangci-lint run ./..."
	@echo "  vet               — go vet ./..."
	@echo "  fmt               — go fmt ./..."
	@echo "  docs              — regenerate $(CLI_DOCS)/ from the live command tree"
	@echo "  release-snapshot  — local goreleaser dry run into ./dist (no push, skips signing)"
	@echo "  release-check     — validate .goreleaser.yaml"
	@echo "  clean             — remove build output"

build:
	$(GO) build -o $(BIN) $(PKG)

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

# docs regenerates the Markdown CLI reference under docs/cli/. The
# hidden `gen-docs` subcommand walks the assembled command tree via
# cobra/doc.GenMarkdownTree, so the output stays in sync with the
# actual --help output. Run `make docs` after touching any flag,
# subcommand registration, or short/long description.
docs:
	rm -rf $(CLI_DOCS)
	mkdir -p $(CLI_DOCS)
	$(GO) run $(PKG) gen-docs $(CLI_DOCS)

# release-snapshot builds every artefact (binaries, deb, rpm, archives,
# checksums) into ./dist without publishing or signing. Use to verify
# .goreleaser.yaml changes locally before tagging.
release-snapshot:
	goreleaser release --snapshot --clean --skip=sign,publish

release-check:
	goreleaser check

clean:
	rm -rf $(BIN) dist/
