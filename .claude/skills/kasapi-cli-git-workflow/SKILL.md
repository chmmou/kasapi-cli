---
name: kasapi-cli-git-workflow
description: Git, commit, and PR workflow for kasapi-cli. TRIGGER on every `git commit` / `git push` / `gh pr create` / `gh pr merge`, when creating a feature or fix branch, on rebase/merge conflicts against `main`, or when the user asks for a code-review pass. Adapted from the libqasapi `qasapi-module` skill, reduced to what applies to a fresh greenfield project.
---

# kasapi-cli Git Workflow

Mandatory workflow for branches, commits, PRs, and review rounds. Applies only to git actions — the domain-specific module slicing from the libqasapi original is **not** carried over.

## Precondition

The repo is currently **not a git repository**. Before the first commit:

```bash
git init -b main
git add <selective: only the files you want in the initial commit>
git commit -m "chore: initial commit"
```

Only after that do the branch/PR rules below apply. The `gh` steps additionally require a configured GitHub remote (`gh repo create …`) — clarify owner / visibility with the user, do not assume.

## Golden Rule

**Keep PR descriptions short.** Summary block only, describing *what*, not *how*. No "Test plan" section, no "🤖 Generated with" footer.

## Conventions

- **Conventional Commits:** `feat:` for new features, `fix:` for bug or schema corrections, `docs:` for documentation- or CHANGELOG-only changes, `chore:` for build/tooling/repo hygiene, `test:` for test-only changes, `refactor:` for structural changes without behavior change.
- **Language:** English commit messages, English CHANGELOG entries, English code comments. Where a file already uses ASCII fallbacks (`ae/oe/ue/ss`), keep that style consistent in the same file; otherwise use real umlauts.
- **Trailer:** **No** `Co-Authored-By: Claude …` trailer in commits. (User-preference override; suppress the Claude Code default template.)
- **Selective `git add`.** Never `git add -A` or `git add .` as a default. Stage only files that belong to the current slice/topic. Excluded by default: `.claude/`, local settings, unrelated fixtures, IDE/OS noise.

## Branch & PR Workflow

```bash
# Create the branch — never commit directly on main
git checkout -b feature/<topic>     # new functionality
git checkout -b fix/<topic>         # bug or schema correction
git checkout -b docs/<topic>        # docs only
git checkout -b chore/<topic>       # tooling / repo hygiene

# Stage + commit
git add <selective>
git commit -m "feat: <short what-sentence>"

# Push + PR
git push -u origin <branch>
gh pr create --base main \
  --title "feat: <short what-sentence>" \
  --body  "<summary block, what only>"
```

PR body: short, describing what was implemented. If an issue is open, reference it with `Closes #<n>` so the project item flips to "Done" automatically on merge.

## CI & Merge

Watch the CI pipeline after the push. Merge only after CI is green:

```bash
gh pr merge <n> --rebase --delete-branch
git checkout main
git pull --ff-only
```

If a rebase merge fails (e.g. because the branch contains a merge commit): fall back to `gh pr merge <n> --squash --delete-branch` with a meaningful subject/body.

## Sandbox Specifics

- **Direct push to `main` is blocked** → always feature branch + PR, no exceptions.
- **`--force-push` is blocked** → on rebase conflicts, integrate via merge: `git fetch origin && git merge origin/main`, resolve conflicts (especially `CHANGELOG.md`) manually, push normally. Never propose `--force` or `--force-with-lease` as a workaround.
- **Do not skip hooks.** No `--no-verify`, no `--no-gpg-sign`. On hook failure, fix the underlying cause and create a **new** commit — do not `--amend`.

## Code-Review Loop

Before any larger new step and after every correction, review the existing code — acting in the role of code reviewer, not implementer. Classify findings:

- **Blocker:** outright wrong (bug, UB, schema mismatch, security) → fix before the next step.
- **Should:** consistency / readability → fix quickly if cheap.
- **Nice-to-have:** cosmetic → record but do not block on it.

Corrected findings are committed on a dedicated `fix/<topic>` branch and merged via a separate PR. After merged corrections, **re-review**, because fixes can introduce regressions — the loop ends only when no blocker or should-findings remain.

## Post-Merge

If a just-merged step has a natural follow-up (a CI flake that surfaced, feature-flag cleanup, triage routine, a "remove once X" TODO), briefly offer a `/schedule` suggestion with a concrete action and cadence. Otherwise skip — refactors, bug fixes with tests, and pure docs PRs do not need a follow-up task.
