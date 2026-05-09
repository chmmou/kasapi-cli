# Security Policy

## Supported Versions

`kasapi-cli` is in pre-1.0 development. Only the latest released
`v0.x.y` minor receives security fixes; older minors are not
backported.

| Version  | Supported          |
| -------- | ------------------ |
| `v0.x.y` (latest) | yes |
| anything else | no  |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Use one of these channels instead:

- GitHub's private vulnerability reporting: open the
  [Report a vulnerability](https://github.com/chmmou/kasapi-cli/security/advisories/new)
  flow under the repository's Security tab.
- E-mail: `developer@kasapi-cli.chm-projects.de`.

Please include:

- a clear description of the issue,
- reproduction steps or a proof-of-concept,
- the affected version (`kasapi-cli --version`),
- your assessment of impact and any suggested mitigation.

## Response Expectations

This is a maintainer-driven hobby project, not a paid product. You
can expect:

- an initial acknowledgement within seven days,
- a follow-up with a triage outcome (accepted / not a vulnerability /
  needs more information) within three weeks,
- a fix landing in a tagged release as soon as practical for accepted
  vulnerabilities, with credit in the release notes unless you prefer
  to remain anonymous.

There is no bug bounty.

## Scope

In scope:

- the `kasapi-cli` binary and its supporting Go packages under
  `internal/`,
- the GitHub Actions workflows under `.github/workflows/`,
- the release artefacts published on the
  [Releases page](https://github.com/chmmou/kasapi-cli/releases) and
  their `cosign` signatures.

Out of scope:

- vulnerabilities in the All-Inkl KAS API itself — please report
  those to All-Inkl directly,
- vulnerabilities in third-party dependencies for which an upstream
  patch already exists; please file or link the upstream advisory
  instead.

## Verifying Releases

Every release artefact and the `SHA256SUMS` file are signed
keylessly with `cosign` via GitHub Actions OIDC. See the
[Install section in `README.md`](README.md#install) for the
verification recipe.
