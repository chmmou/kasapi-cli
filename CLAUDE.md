# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository State

The read-phase modules are wired up (accounts, server, domains/subdomains/TLDs, DNS, mail, databases, FTP/Samba users, cronjobs, directory protection, software installs, DDNS users, usage statistics), and so are most v0.2.0 write slices (mail accounts/forwards/filters/lists, databases, FTP/Samba users, DDNS users, cronjobs, directory protection) including the destructive-write safety contract (confirmation gate, `--dry-run`, audit records — see `docs/usage/destructive-writes.md`). The CI gate (`lint & test`, `docs sync`, `goreleaser` config check, `govulncheck`, CodeQL) runs on every push and pull request. `main` is protected: signed commits are required, force-pushes to `main` are blocked for the GitHub UI / `gh pr merge --rebase` (signatures are stripped server-side), and merging happens via a locally-rebased fast-forward push by the maintainer (see `.claude/skills/kasapi-cli-git-workflow/SKILL.md`).

The remaining write endpoints (software installs, filesystem/SSL helpers) are part of the v0.2.0 backlog tracked on the *kasapi-cli v0.1.0* GitHub project; do not invent endpoints not documented at <https://kasapi.kasserver.com/dokumentation/phpdoc/>.

There is no predecessor library and no inherited backlog. Do not assume or import patterns from any other KAS client; design from the KAS API docs and the fixtures in `testdata/`.

## What kasapi-cli Is

`kasapi-cli` is a CLI for the **All-Inkl KAS API** (Kunden-Administrations-System — a SOAP/XML hosting-control API). The only external references for the API surface are:

- The two KAS endpoint URLs (auth and API), to be supplied by the user / config.
    - api: https://kasapi.kasserver.com/soap/KasApi.php
    - auth: https://kasapi.kasserver.com/soap/KasAuth.php
- The KAS API documentation: <https://kasapi.kasserver.com/dokumentation/phpdoc/>. Per-function pages live under `files/<kas_action>-inc.html` (e.g. `files/get-accounts-inc.html`).

Treat that documentation as the contract for request shapes; treat `testdata/*.xml` as the contract for response shapes.

## testdata/

`testdata/<module>/*.xml` are **real KAS API responses** captured for offline parser/mapping tests. They are the source of truth for response shape — when a mapping test fails, suspect the mapping before the fixture.

Filename convention:

- Subdirectory per module: `testdata/<module>/`, e.g. `testdata/account/`, `testdata/mailinglist/`. Shared cross-module fault fixtures (`response_failed_no_auth.xml`, `response_failed_kas_session_invalid.xml`, ...) live at the top of `testdata/`.
- One file per `(kas_action, kind)` pair: `<kas_action>_<kind>[_<variant>].xml`, where `kind` is `request` or `response_<status>`, and `status` is `success` or `failed`. Examples:
    - `get_accounts_response_success.xml`
    - `add_account_response_failed_account_kas_password_syntax_incorrect.xml`
    - `get_ftpuser_response_success_empty_list.xml` (variant of the success shape)

Add a new fixture whenever a new KAS call is wired up; redact secrets before committing.

## Authoritative Style & Architecture

Go style, architecture, patterns, and linting rules for this repo live in:

- `AGENTS.md` — top-level operating rules (always run `gofmt`/`goimports`, `go vet`, `golangci-lint`, `go test`; clean-architecture layering; no business logic depending on transport).
- `docs/go/STYLE_GUIDE.md`
- `docs/go/ARCHITECTURE.md` — clean-architecture layering: `cmd/kasapi-cli` wires; domain/use cases in `internal/<domain>/`; SOAP/HTTP/CLI are outer-layer adapters.
- `docs/go/PATTERNS.md`
- `docs/go/LINTING.md` — CI gate set.

Project-specific operating rules (git/PR mechanics, code-review classification, vertical-slice pattern) live alongside the Go references:

- `.claude/skills/kasapi-cli-git-workflow/SKILL.md` — branches, signed commits, FF-push merge model, no `Co-Authored-By` trailer.
- `.claude/skills/kasapi-cli-code-review/SKILL.md` — Blocker / Should / Nice-to-have classification, re-review cycle.
- `.claude/skills/kasapi-cli-vertical-slice/SKILL.md` — slice anatomy + order of operations per KAS endpoint (fixture → mapping → client → CLI → docs → CHANGELOG).
- `CONTRIBUTING.md` — vertical-slice pattern per KAS endpoint, language conventions, security reporting.

Read these before designing or extending package layout. Do not duplicate their content into new docs — link to them.

## Commands

The standard loop, available both as raw `go` invocations and as `make` targets defined in the top-level `Makefile`:

```sh
go fmt ./...                          # or: make fmt
go vet ./...                          # or: make vet
golangci-lint run ./...               # or: make lint
go test ./...                         # or: make test
go test -race ./...                   # for packages with concurrency
go test ./internal/<pkg> -run TestXxx # single test
go build ./cmd/kasapi-cli             # or: make build
make docs                             # regenerate docs/cli/ from the live command tree
```

The `docs sync` CI job runs `make docs` and fails when the checked-in `docs/cli/` differs from the regenerated output, so a flag, subcommand registration, or short/long-description change must come paired with a `make docs && git add docs/cli/` step.
