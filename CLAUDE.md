# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository State

This is a **new, greenfield project**:

- No Go source tree yet (`cmd/`, `internal/`, `go.mod` are not present).
- Not yet a git repository — `git init` is required before the first feature branch / PR.
- Only docs, agent guidance, and KAS-API XML fixtures are checked in.

When asked to "start" or "bootstrap", expect to be the one creating the module layout from scratch following `docs/go/ARCHITECTURE.md`.

There is no predecessor library and no inherited backlog. Do not assume or import patterns from any other KAS client; design from the KAS API docs and the fixtures in `testdata/`.

## What kasapi-cli Is

`kasapi-cli` is a CLI for the **All-Inkl KAS API** (Kunden-Administrations-System — a SOAP/XML hosting-control API). The only external references for the API surface are:

- The two KAS endpoint URLs (auth and API), to be supplied by the user / config.
    - api: https://kasapi.kasserver.com/soap/KasApi.php
    - auth: https://kasapi.kasserver.com/soap/KasAuth.php
- The KAS API documentation: <https://kasapi.kasserver.com/dokumentation/phpdoc/>.

Treat that documentation as the contract for request shapes; treat `testdata/*.xml` as the contract for response shapes.

## testdata/

`testdata/*.xml` are **real KAS API responses** captured for offline parser/mapping tests. They are the source of truth for response shape — when a mapping test fails, suspect the mapping before the fixture. Filenames follow the KAS function name (`get_<thing>.xml`; `get_<thing>_with_param.xml` for parameterized variants). Add a new fixture whenever a new KAS call is wired up.

## Authoritative Style & Architecture

Go style, architecture, patterns, and linting rules for this repo live in:

- `AGENTS.md` — top-level operating rules (always run `gofmt`/`goimports`, `go vet`, `golangci-lint`, `go test`; clean-architecture layering; no business logic depending on transport).
- `docs/go/STYLE_GUIDE.md`
- `docs/go/ARCHITECTURE.md` — clean-architecture layering: `cmd/kasapi-cli` wires; domain/use cases in `internal/<domain>/`; SOAP/HTTP/CLI are outer-layer adapters.
- `docs/go/PATTERNS.md`
- `docs/go/LINTING.md` — CI gate set.

Read these before designing or extending package layout. Do not duplicate their content into new docs — link to them.

## Commands

Once `go.mod` exists, the standard loop is:

```sh
go fmt ./...
go vet ./...
golangci-lint run ./...
go test ./...
go test -race ./...                     # for packages with concurrency
go test ./internal/<pkg> -run TestXxx   # single test
go build ./cmd/kasapi-cli
```

There is no build/test runnable yet — running these in the current tree will fail until the module is bootstrapped.
