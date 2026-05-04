---
name: kasapi-cli-code-review
description: Code-review loop for kasapi-cli — classification of findings (Blocker/Should/Nice-to-have) and the re-review cycle that ends only when no Blocker or Should-finding remains. TRIGGER when the user asks for a code-review pass, before any larger new step, or after corrections have been merged. The git/PR mechanics (branch naming, signed FF-push, PR shape) live in the companion skill `kasapi-cli-git-workflow`.
---

# kasapi-cli Code-Review Loop

Use this skill whenever the user asks for a review pass, before starting a larger new step, or directly after corrections have been merged. The git mechanics for landing the resulting fix branch (selective `git add`, signed commits, FF-push to `main`, branch cleanup) are **not** repeated here — defer to the companion skill **`kasapi-cli-git-workflow`** for those.

## Stance

Act in the role of code reviewer, not implementer. Read the diff or the affected code path with fresh eyes; resist the temptation to silently fix what you find while reading. Surface findings first, decide together what to address, then switch back to implementer mode on a dedicated branch.

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
