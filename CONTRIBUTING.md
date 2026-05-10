# Contributing to kasapi-cli

Thanks for your interest in contributing. This document collects the conventions a contributor needs to know on top of the in-tree references.

## Before you start

- This project is **not affiliated with All-Inkl.com**. The KAS-API surface is reverse-engineered from the public documentation at <https://kasapi.kasserver.com/dokumentation/phpdoc/> and from real responses captured in `testdata/`.
- The roadmap in [README.md](README.md#roadmap) shows which endpoints are already wired up. If you want to claim an unchecked item, please open an issue first so work is not duplicated.
- Bug reports and PRs that touch the KAS-API contract should reference the relevant doc page and, where possible, attach a redacted response fixture.

## Authoritative references

Read these before designing changes — do not duplicate their content into new docs:

- [`AGENTS.md`](AGENTS.md) — top-level operating rules: always run `gofmt`/`goimports`, `go vet`, `golangci-lint`, `go test`; clean-architecture layering; no business logic depending on transport.
- [`docs/go/ARCHITECTURE.md`](docs/go/ARCHITECTURE.md) — `cmd/kasapi-cli` wires; domain/use cases in `internal/<domain>/`; SOAP/HTTP/CLI are outer-layer adapters.
- [`docs/go/STYLE_GUIDE.md`](docs/go/STYLE_GUIDE.md) — Go style.
- [`docs/go/PATTERNS.md`](docs/go/PATTERNS.md) — recurring patterns (e.g. the per-package `Caller` interface, fixture-backed decoders, `cli.Tabular`).
- [`docs/go/LINTING.md`](docs/go/LINTING.md) — the CI gate set.
- [`.claude/skills/kasapi-cli-git-workflow/SKILL.md`](.claude/skills/kasapi-cli-git-workflow/SKILL.md) — git, branch, signed-commit, PR, and FF-push merge mechanics enforced for this project.
- [`.claude/skills/kasapi-cli-code-review/SKILL.md`](.claude/skills/kasapi-cli-code-review/SKILL.md) — code-review loop (Blocker/Should/Nice classification, re-review cycle).

## Repository layout

- `cmd/kasapi-cli/` — CLI entry point.
- `internal/` — domain types, transport, mappers, CLI wiring; one package per KAS resource (see `internal/account/`, `internal/domain/`, `internal/dns/`, …) plus shared infrastructure (`internal/soap`, `internal/api`, `internal/auth`, `internal/transport`, `internal/session`, `internal/config`, `internal/cli`).
- `testdata/` — recorded KAS-API SOAP responses; the source of truth for response shape, used by offline parser tests.
- `docs/go/` — Go style, architecture, patterns, and linting reference for this repo.
- `.claude/skills/kasapi-cli-git-workflow/` — git/PR/merge mechanics enforced for this project.
- `.claude/skills/kasapi-cli-code-review/` — code-review loop (Blocker/Should/Nice classification, re-review cycle).
- `AGENTS.md`, `CLAUDE.md` — agent guidance.

## Development setup

Prerequisites:

- Go (latest stable; the version pinned in `go.mod`).
- [`golangci-lint`](https://golangci-lint.run/) matching the version used in CI.
- A working GnuPG key — commits to `main` must be signed (the branch is protected with `required_signatures`).

Standard loop:

```sh
go fmt ./...
go vet ./...
golangci-lint run ./...
go test ./...
go test -race ./...                     # for packages with concurrency
go build ./cmd/kasapi-cli
```

## Vertical-slice pattern

Each KAS endpoint is added as a single vertical slice; do not split it across modules. A new endpoint typically needs:

1. **Type & decoder** in `internal/<domain>/<name>.go` — typed Go value + `Decode<Thing>` mapping the SOAP `ns2:Map` / `Array` payload, plus a per-package `Caller` interface so tests do not need network setup.
2. **Client method** on the package's `*Client` (e.g. `(c *Client) Get(ctx, name)`).
3. **Test** in `internal/<domain>/<name>_test.go` — fixture-backed mapping tests + a `fakeCaller` for the client method (assert the action name and any params).
4. **Fixture** in `testdata/<domain>/<kas_action>_response_success.xml` (and `_request.xml` if useful). The fixtures are real captured responses with secrets redacted — they are the source of truth for response shape.
5. **CLI subcommand** in `internal/cli/<domain>.go`, registered in `cmd/kasapi-cli/main.go`.
6. **CHANGELOG entry** under `## [Unreleased] / ### Added` (or `### Fixed`, etc.). One paragraph, ending with `Closes #<issue>` if applicable.
7. **Roadmap update** in [README.md](README.md#roadmap) — flip the corresponding `- [ ]` to `- [x]`.

When the response shape differs between list and singular views (e.g. `get_domains` with vs. without `domain_name`), prefer one struct with `omitempty` on the view-specific fields rather than two structs.

## Language

- Commits, CHANGELOG entries, code comments, doc.go strings, and PR text are in **English**.
- Where a file already uses ASCII fallbacks (`ae/oe/ue/ss`), keep that style consistent in the same file; otherwise use real umlauts.

## Commit conventions

- **[Conventional Commits](https://www.conventionalcommits.org/):** `feat:` for new features, `fix:` for bug or schema corrections, `docs:` for documentation- or CHANGELOG-only changes, `chore:` for build/tooling/repo hygiene, `test:` for test-only changes, `refactor:` for structural changes without behavior change.
- **Signed commits.** Do not skip hooks (no `--no-verify`, no `--no-gpg-sign`). On a hook failure, fix the underlying cause and create a **new** commit — do not `--amend`.
- **No `Co-Authored-By` trailer.** Maintainers do not use the Claude Code default trailer; please omit it as well.
- **Selective `git add`.** Stage only files that belong to the current slice; never `git add -A` or `git add .`. Excluded by default: `.claude/`, local settings, unrelated fixtures, IDE/OS noise, secrets, captured KAS responses that have not been redacted.

## Branches and pull requests

```sh
git checkout -b feature/<topic>     # new functionality
git checkout -b fix/<topic>         # bug or schema correction
git checkout -b docs/<topic>        # docs only
git checkout -b chore/<topic>       # tooling / repo hygiene
```

PR body: keep it short. Summary block describing *what*, not *how*. No "Test plan" section, no generated-by footer. If an issue is open, reference it with `Closes #<n>` so the project item flips to "Done" automatically on merge.

CI must be green before merge. The CI workflow runs `go fmt` (check-only), `go vet`, `golangci-lint`, and `go test` (with `-race` where applicable).

`main` is protected with `required_signatures` + `enforce_admins` + `linear_history`. The GitHub UI / `gh pr merge` strips signatures and is therefore **not** used for this repo. Merging is done by the maintainer via a locally-rebased, signed fast-forward push to `main`. As a contributor you do not need to do this — you only need to keep your branch rebased on `main` and your commits signed.

## Code review

Findings are classified:

- **Blocker** — outright wrong: bug, undefined behavior, KAS-API schema mismatch, security issue, broken invariant. Must be fixed before merge.
- **Should** — consistency / readability / small footguns. Fix quickly if cheap; otherwise capture as a follow-up issue.
- **Nice-to-have** — cosmetic, style, or speculative. Recorded as a grouped follow-up issue but not a merge blocker.

Corrections from review land on a dedicated `fix/<topic>` branch via a separate PR; the loop ends only when no Blocker or Should-finding remains.

## Releases

Before a release tag is pushed, run a local goreleaser snapshot to validate the pipeline against your branch:

```sh
make release-snapshot   # goreleaser release --snapshot --clean --skip=sign,publish
```

The snapshot exercises the same build/archive/package steps as the real release without producing signatures or publishing to GitHub. Inspect at least one of the resulting `dist/kasapi-cli_*_linux_amd64.tar.gz` archives — `tar -tzf` should list `LICENSE`, `README.md`, `CHANGELOG.md`, and the rendered docs under `docs/cli/` and `docs/usage/` alongside the binary. If the docs are missing, an `archives.files` glob in `.goreleaser.yaml` is silently failing to match (this caught a `**/*` regression on the v0.1.0-alpha.1 cut). The snapshot also catches `goreleaser check` config errors and any build-matrix mismatch.

`make release-snapshot` requires `goreleaser` (and `syft`, if SBOM generation should also be exercised locally — append `,sbom` to the `--skip` list if syft is not installed).

## Reporting security issues

Please do **not** open a public issue for security vulnerabilities. Use the [private vulnerability reporting flow](https://github.com/chmmou/kasapi-cli/security/advisories/new) under the repository's Security tab, or e-mail the maintainer. Full policy in [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the project's [BSD 3-Clause License](LICENSE).
