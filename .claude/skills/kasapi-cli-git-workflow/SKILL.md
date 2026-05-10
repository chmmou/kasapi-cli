---
name: kasapi-cli-git-workflow
description: Git, commit, branch, and PR/merge mechanics for kasapi-cli. TRIGGER on every `git commit` / `git push` / `gh pr create`, when creating a feature or fix branch, on rebase/merge conflicts against `main`, or on the signed-commit fast-forward merge to `main`. For the review-classification loop see the companion skill `kasapi-cli-code-review`.
---

# kasapi-cli Git Workflow

Mandatory mechanics for branches, commits, PRs, and the merge to `main`. The review-loop side of the workflow (Blocker/Should/Nice classification, re-review after corrections) lives in the companion skill **`kasapi-cli-code-review`** — invoke that one when the user asks for a review pass or you are about to start a larger new step.

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

Watch the CI pipeline after the push. Merge only after CI is green.

`main` has GitHub branch protection with **require signed commits** + **enforce admins** + **linear history**. The GitHub UI / `gh pr merge --rebase` button creates a server-side commit that is **not** signed by your key — that merge will be rejected. Therefore: rebase locally (preserving your signature) and fast-forward `main` directly.

```bash
# Sync main and rebase the feature branch on top of it.
git fetch origin
git checkout <branch>
git rebase origin/main

# Update the PR with the rebased branch (force-with-lease is fine on feature branches).
git push --force-with-lease origin <branch>

# Fast-forward main to the rebased tip (preserves your signed commits).
git push origin <branch>:main

# Cleanup. The PR auto-closes because its commits are now in main.
git checkout main
git pull --ff-only
git branch -D <branch>
git push origin --delete <branch>
```

`gh pr merge` is **not used** for this repo — it always strips signatures.

## Sandbox Specifics

- **Direct push to `main`** is normally blocked by the harness → each push needs explicit per-instance authorization. This is the documented merge path; obtain authorization rather than working around it.
- **`--force-push` is blocked** by default; same authorization pattern. Never bypass with `-c` overrides or `--no-verify`.
- **Do not skip hooks.** No `--no-verify`, no `--no-gpg-sign`. On hook failure, fix the underlying cause and create a **new** commit — do not `--amend`.
- **Branch protection on `main`** (settings: `required_signatures`, `enforce_admins`, `required_linear_history`, `allow_force_pushes` kept on for retroactive history fixes, `allow_deletions` off). Pushes to `main` of unsigned commits will be rejected by GitHub.

## Release tags

Before pushing a `vX.Y.Z` tag that triggers `release.yml`, run the local goreleaser snapshot on the would-be release commit and verify the resulting archives contain everything the workflow will publish:

```sh
make release-snapshot                                       # = goreleaser release --snapshot --clean --skip=sign,publish
tar -tzf dist/kasapi-cli_*_linux_amd64.tar.gz               # expect: LICENSE, README.md, CHANGELOG.md, docs/cli/*, docs/usage/*, kasapi-cli
```

Reason: the v0.1.0-alpha.1 cut surfaced two pipeline bugs that **the real workflow could not have caught quickly** — an `archives.files` glob (`docs/cli/**/*` matched zero entries because the directory is flat) and an action-version mismatch in `release.yml`. The snapshot reproduces every goreleaser step short of sign/publish, so silent glob failures and `goreleaser check` config errors land in your terminal instead of as a failed workflow run with a now-burned tag.

How to apply: snapshot must succeed and the archive listing must include the expected docs files before you create the tag. If the runner has no `syft` installed, append `,sbom` to the `--skip` list (`--skip=sign,publish,sbom`) — that does not exercise SBOM generation but does not block the rest of the pre-check. Production SBOMs come from CI's `anchore/sbom-action/download-syft` step, which is independent.

## Post-Merge

If a just-merged step has a natural follow-up (a CI flake that surfaced, feature-flag cleanup, triage routine, a "remove once X" TODO), briefly offer a `/schedule` suggestion with a concrete action and cadence. Otherwise skip — refactors, bug fixes with tests, and pure docs PRs do not need a follow-up task.
