<!--
PR title: use Conventional Commits — feat:/fix:/docs:/chore:/refactor:/test:.
Keep this body short. Summary block describing *what*, not *how*.
No "Test plan" section. No generated-by footer. No Co-Authored-By trailer.
-->

## Summary

<!-- One or two sentences. What changed. -->

Closes #

## KAS-API surface (if applicable)

<!-- Which kas_action(s) does this PR cover? Skip if N/A (docs / chore / infra). -->

- Action(s):
- Doc page:

## Vertical-slice checklist

<!-- For new endpoints. Tick what applies; remove the section for pure docs/chore PRs. See CONTRIBUTING.md. -->

- [ ] Type & decoder in `internal/<domain>/<name>.go`
- [ ] Client method on the package's `*Client`
- [ ] Fixture-backed test in `internal/<domain>/<name>_test.go`
- [ ] Captured response under `testdata/<domain>/<kas_action>_response_*.xml` (redacted)
- [ ] CLI subcommand wired into `cmd/kasapi-cli/main.go`
- [ ] CHANGELOG entry under `## [Unreleased]`
- [ ] [ROADMAP.md](../ROADMAP.md) updated (`- [ ]` → `- [x]`)

## Author confirmation

- [ ] Commits are signed and follow Conventional Commits (`feat:`, `fix:`, …).
- [ ] No credentials, session tokens, or unredacted account data in any fixture, log line, or test.
- [ ] `go fmt`, `go vet`, `golangci-lint`, and `go test` (with `-race` where applicable) pass locally.
