---
name: kasapi-cli-code-review
description: Code-review loop for kasapi-cli — kasapi-cli-specific review anchors (fixture ↔ mapping alignment, clean-architecture layering, vertical-slice completeness, stable error identifiers), classification of findings (Blocker/Should/Nice-to-have), the batched fix pass that lands all agreed findings of one review pass as one commit on one fix/* branch (with a pre-commit fresh-eyes diff review), and the re-review cycle that ends only when no Blocker or Should-finding remains. Supports running the loop as dedicated fresh-eyes sub-agents (Reviewer → Fixer → Re-Reviewer) so no agent reviews its own work. TRIGGER when the user asks for a code-review pass, asks what to look for in a diff, when executing the agreed findings of a review pass, before any larger new step, or after corrections have been merged. The git/PR mechanics (branch naming, signed FF-push, PR shape) live in the companion skill `kasapi-cli-git-workflow`.
---

# kasapi-cli Code-Review Loop

Use this skill whenever the user asks for a review pass, before starting a larger new step, or directly after corrections have been merged. The git mechanics for landing the resulting fix branch (selective `git add`, signed commits, FF-push to `main`, branch cleanup) are **not** repeated here — defer to the companion skill **`kasapi-cli-git-workflow`** for those.

## Stance

Act in the role of code reviewer, not implementer. Read the diff or the affected code path with fresh eyes; resist the temptation to silently fix what you find while reading. Surface findings first, decide together what to address, then switch back to implementer mode on a dedicated branch.

"Fresh eyes" is strongest when the reviewer is a *different* agent than the implementer, and the agent that re-reviews a fix is *different* than the one that wrote it. When the surface is broad or the user asks for a delegated review, run the loop as separate sub-agents per the **Delegated Multi-Agent Loop** below rather than switching hats in one context.

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

## Batched Fix Pass (executing the agreed findings)

Once the user has chosen which findings to address, execute them as **one batched fix pass** — the shape used for the second/third/fourth whole-codebase passes:

- **One branch, one commit round.** All agreed Blocker/Should (and explicitly approved Med/Low) findings of a pass land together on a single `fix/<topic>` branch and are squashed into **one** signed commit — no per-finding commits. Nice-to-haves are never fixed in the pass; they go to the grouped follow-up issue (below).
- **Order of operations:**
  1. Branch from the current `main`; read every affected file before editing.
  2. While implementing, re-verify each finding against its source of truth (KAS API documentation for request shapes, `testdata/` fixtures for response shapes) — a review finding can itself be wrong.
  3. Implement each fix together with a test that pins the fixed behavior; update any doc whose claim the fix changes (`docs/usage/`, `CLAUDE.md`, CHANGELOG).
  4. Full local gate: `go fmt` / `go vet` / `golangci-lint run` / `go test ./...`, plus `make docs` whenever a command, flag, or help text changed.
  5. **Binary verification** for CLI-behavior findings: build the binary into a scratch directory (never the repo root) and probe the changed behavior directly — exit codes, help output, rendered messages — instead of trusting unit tests alone.
  6. **Pre-commit fresh-eyes diff review**: before the commit, spawn two parallel *read-only* reviewer sub-agents over the *uncommitted* diff — one with a correctness/regression lens (library semantics, edge cases, side effects), one with a rules/docs/convention lens (project + global rules, CHANGELOG accuracy, fixture naming, help-text conventions). The conductor independently verifies decisive claims. Blocker/Should findings are fixed before committing; trivial cosmetic points may be applied inline, everything else joins the grouped-issue comment. Reviewing *before* the single commit exists because `--amend` is forbidden in this repo — it is the only way to keep the pass at one commit.
  7. One selective `git add` + one signed commit, push, PR, CI green, then the FF-merge and branch cleanup per `kasapi-cli-git-workflow` (the push to `main` always needs the user's per-instance authorization).
- **CHANGELOG shape:** one umbrella bullet per pass under `[Unreleased]/Fixed` — `Whole-codebase re-review follow-up (<n>th pass, <severities>)` — with one sub-bullet per finding, inserted above the previous pass's entry (newest first).

## Delegated Multi-Agent Loop (fresh eyes)

When the review surface is broad (multi-file, cross-module, a whole package or branch) or the user asks for a delegated review, run the loop as a chain of dedicated, freshly-spawned sub-agents instead of switching hats in one context. The point is structural independence: no agent ever reviews its own work.

Each leg is a separate sub-agent, spawned per pass and given only the structured hand-off (not the others' reasoning):

- **Reviewer agent** — read-only. Produces the findings report grouped by classification (the *Reporting Findings* shape). Must not edit anything.
- **Conductor** (the orchestrating main thread) — independently verifies the decisive findings before acting on them (a wrong finding must not drive a wrong fix), classifies, and obtains the user's scope decision per *Reporting Findings*. The conductor is never delegated: gated or irreversible steps — the signed FF-push to `main`, branch cleanup — stay with the conductor (see `kasapi-cli-git-workflow`) and are never handed to a sub-agent.
- **Fixer agent** — implements only the agreed items on a dedicated `fix/<topic>` branch, takes the full local gate green, opens the PR. Does **not** merge to `main`.
- **Re-Reviewer agent** — a *fresh* agent (never the Fixer) that reviews the fix diff specifically hunting fix-introduced regressions, and ends with an explicit verdict: `no Blocker/Should → loop may close`, or the new findings.

Loop: a Re-Reviewer that finds a Blocker/Should → spawn a new Fixer agent, then a new Re-Reviewer agent; repeat. The loop ends only when a Re-Reviewer returns the close verdict — the same termination rule as the inline cycle.

Hand-offs between legs are structured, not transcripts: the Reviewer passes the `file:line` + classification report; the Fixer passes `git diff --stat` + PR ref + any test/string adjustments and flagged out-of-band drift; the Re-Reviewer passes the verdict + any new findings. Keep each agent's brief scoped to its own leg so its eyes stay fresh.

Don't over-orchestrate: a single small diff stays an inline review — the delegated chain is for breadth or when the user asks for it.

## Follow-Up Issues

When recording the Nice-to-have bundle on GitHub:

- **reuse the existing grouped NTH issue when one is open** (e.g. #200): append one comment per review pass, prefixed with which pass and date it came from — do not open a new issue per pass; only open a new grouped issue when none exists,
- bullet list with the same `file:line` + one-sentence-each shape used in the report,
- cosmetic-only findings surfaced by the pre-commit diff review of the fix branch join the same comment as a clearly-marked addendum,
- mark items that overlap an already-tracked class in the issue as such instead of re-filing them,
- attach a newly-created issue to the active project board so it is tracked alongside feature work; label `documentation` for doc/comment-only items, otherwise leave unlabeled and let triage decide.
