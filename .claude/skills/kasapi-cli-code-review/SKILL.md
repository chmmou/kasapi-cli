---
name: kasapi-cli-code-review
description: Code-review loop for kasapi-cli — kasapi-cli-specific review anchors (fixture ↔ mapping alignment, clean-architecture layering, vertical-slice completeness, stable error identifiers), classification of findings (Blocker/Should/Nice-to-have), and the re-review cycle that ends only when no Blocker or Should-finding remains. TRIGGER when the user asks for a code-review pass, asks what to look for in a diff, before any larger new step, or after corrections have been merged. The git/PR mechanics (branch naming, signed FF-push, PR shape) live in the companion skill `kasapi-cli-git-workflow`.
---

# kasapi-cli Code-Review Loop

Use this skill whenever the user asks for a review pass, before starting a larger new step, or directly after corrections have been merged. The git mechanics for landing the resulting fix branch (selective `git add`, signed commits, FF-push to `main`, branch cleanup) are **not** repeated here — defer to the companion skill **`kasapi-cli-git-workflow`** for those.

## Stance

Act in the role of code reviewer, not implementer. Read the diff or the affected code path with fresh eyes; resist the temptation to silently fix what you find while reading. Surface findings first, decide together what to address, then switch back to implementer mode on a dedicated branch.

## Review Anchors (kasapi-cli-specific)

Go-level hygiene (DRY, error-wrapping convention, type discipline, exhaustiveness) is the domain of `docs/go/STYLE_GUIDE.md`, `docs/go/PATTERNS.md`, and `docs/go/LINTING.md` — and `golangci-lint` enforces most of it mechanically. Do **not** re-derive those rules here; trust the lint gate, then focus the review on anchors unique to this repo:

- **Fixture ↔ mapping alignment** — `testdata/<module>/*.xml` is captured from the live API and is the source of truth for response shape. When a mapping test fails, suspect the mapping before the fixture; when a mapping change is proposed without a touching fixture, flag it.
- **Clean-architecture layering** — domain code in `internal/<domain>/` must not import the outer-layer adapters (SOAP, HTTP, Cobra/CLI); conversely the outer-layer packages (`internal/soap/`, `internal/transport/`, `internal/cli/`, `cmd/kasapi-cli/`) carry no domain logic. See `docs/go/ARCHITECTURE.md` for the full split.
- **Vertical-slice completeness** — every new endpoint touches fixture → mapping (+ test) → client accessor → CLI subcommand → `docs/cli/` regen → CHANGELOG entry. A missing leg is a Should-finding by default; a missing fixture or test is a Blocker.
- **Stable error identifiers** — where a caller needs to branch on a failure (auth refresh, flood-protection fallback, `ErrNoConfig`), the error must carry a typed sentinel or `errors.Is`-able marker, not a string match.

## Classification

Every finding is one of:

- **Blocker** — outright wrong: bug, undefined behavior, schema mismatch with the KAS API, security issue, broken invariant. Fix before the next step.
- **Should** — consistency / readability / small footguns. Fix quickly if cheap; otherwise capture as a follow-up issue.
- **Nice-to-have** — cosmetic, style, or speculative. Record (typically as a single grouped follow-up issue attached to the project) but do not block on it.

When unsure between Blocker and Should, prefer Blocker — the cost of an extra fix PR is lower than the cost of shipping a wrong behavior.

## Reporting Findings

Group the report by classification, not by file. For each finding give:

- file path with line number (`internal/api/client.go:43`),
- one sentence on what is wrong,
- one sentence on the suggested direction (not the full patch).

Then ask the user which items to address now versus file as follow-ups. Do not start editing before the user has chosen.

## Correction & Re-Review Cycle

Corrections land on a dedicated `fix/<topic>` branch via a separate PR — see `kasapi-cli-git-workflow` for the branch/PR/merge mechanics. After the corrections merge, **re-review** the touched area, because fixes can introduce regressions. The loop ends only when no Blocker or Should-finding remains; Nice-to-haves are captured as a single grouped issue attached to the project board and explicitly out of scope for the current loop.

## Follow-Up Issues

When filing the Nice-to-have bundle as a GitHub issue:

- one issue per review pass, not one per finding,
- bullet list with the same `file:line` + one-sentence-each shape used in the report,
- attach to the active project board so the item is tracked alongside feature work,
- label `documentation` for doc/comment-only items, otherwise leave unlabeled and let triage decide.
