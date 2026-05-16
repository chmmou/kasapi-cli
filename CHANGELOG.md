# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `kasapi-cli sessions delete` invalidates the resolved profile's
  cached session token both server-side (KAS API `delete_session`) and
  in the local `sessions.toml` cache. It acts on the *currently
  cached* token only and never bootstraps a fresh one just to delete
  it. The command is idempotent: a missing or already-invalid
  (`unknown_session`) session is reported and exits 0; any other
  transport/KAS error is surfaced with a non-zero exit after the local
  cache has been cleared (the local cache is the authoritative
  client-side state). No confirmation prompt — deleting a session
  merely forces a re-authentication on the next session-mode call.
  `add_session` is *not* a separate endpoint: it is the KasAuth
  credential-token flow already covered by `internal/auth`, so it gets
  no subcommand. The shared `delete_session` use case now lives in
  `internal/session` (`session.Client`); `config use-profile`'s
  revoke path was refactored to delegate to it. Closes #60.

- `kasapi-cli config add-profile <name>` / `use-profile <name>` /
  `list-profiles` for managing multiple profiles in `config.toml`.
  `add-profile` reuses the `config init` prompt flow with `--force` to
  overwrite an existing profile; `use-profile` flips `default_profile`
  and, if the outgoing profile had a non-expired cached session token,
  invokes the KAS API `delete_session` action on it before removing
  the local entry from `sessions.toml`. The server-side revoke is
  best-effort — a transport or `unknown_session` fault is logged
  under `--verbose` but does not abort the switch, because the local
  cache is the authoritative client-side state. `list-profiles` prints
  one line per profile, marks the default with `* `, and never writes
  `auth_data`. Closes #39.

### Changed

- Strict period decoding in `get_traffic` (re-review follow-up):
  `usage.DecodeTraffic` now decodes the mandatory `year` and `month`
  fields with `soap.Value.MapIntStrict`, so a missing or non-numeric
  value yields a decode error instead of a silent `0` that would
  mislabel the reporting period. Scope is deliberately limited to
  those two always-present fields: `day` stays lenient (the monthly
  summary row legitimately omits it) and the `http_`/`ftp_` traffic
  and hit counters stay lenient (KAS returns `xsi:nil` for a bucket
  with no data, so a strict reading would turn a real no-traffic
  response into a hard error). This continues the strict-numeric
  rollout deferred in the first review follow-up.

- Strict numeric decoding for required KAS fields (review follow-up):
  added `soap.Value.AsIntStrict` / `MapIntStrict` / `MapInt64Strict`,
  the error-returning siblings of the lenient `AsInt` / `MapInt` /
  `MapInt64` accessors — a missing key, empty/`xsi:nil`, or non-numeric
  value yields an error instead of being silently coerced to `0`;
  `MapIntStrict` additionally rejects a value that would overflow the
  platform `int` (32-bit targets) rather than truncating it silently.
  Adopted in the account resource-quota decoder (`decodeQuota`): a
  present quota Map whose mandatory `max`/`reserved`/`created`/`used`/
  `free` integers are malformed now fails `DecodeAccountResources`
  rather than misreporting a `0` limit. The lenient accessors are
  unchanged and remain the right choice for genuinely optional fields;
  broader per-field adoption is deferred until each field's
  presence is fixture-verified.

- Repo-consistency pass (post-review): `testdata/statistic/` renamed to
  `testdata/usage/` so the fixture subdirectory matches its module
  (`internal/usage`, CLI `usage`) per the one-subdir-per-module
  convention; `TestServerCmdHelpListsInfo` moved out of
  `account_test.go` into its own `internal/cli/server_test.go`;
  `internal/cli/gendocs_test.go` added to cover the `gen-docs` helper;
  `internal/{ssl,chown,symlink}` doc comments now state explicitly that
  the packages are not-yet-implemented placeholders (issue #13); a
  comment on `api.CodeUnknownAction` records that `"unkown_action"`
  mirrors the KAS API's own misspelling verbatim and must not be
  "corrected". No behaviour change.

- `internal/cli/wire.go`: the first-run error from `BuildAPIClient` now
  appends ``(run `kasapi-cli config init` to create a profile
  interactively)`` when no config file exists at all, so a fresh user
  who runs `kasapi-cli accounts list` without env vars or flags is
  pointed at the existing bootstrap wizard instead of having to guess
  at flag combinations. Partial-config cases (file exists but a profile
  is incomplete) keep the bare validation error because `config init`
  refuses to overwrite without `--force` in that scenario.

- `internal/session/store.go` + `auth/source.go` + `api/client.go`:
  thread the call's `context.Context` through `session.Store.Load` /
  `Save` / `Delete` and through `api.Heartbeater.Heartbeat`. The lock
  wait now uses `flock.TryLockContext` so a user `Ctrl-C` while
  another `kasapi-cli` process holds the sessions-file lock aborts
  cleanly instead of blocking forever; the synchronous toml/os calls
  are local-FS only so a `ctx.Err()` check at the boundary is
  sufficient. `SessionTokenSource.Invalidate` deliberately uses
  `context.Background` for the delete so a cancelled run still clears
  the stale on-disk token, matching the "cleanup is finalisation"
  pattern. Tests updated; behaviour for non-cancelled calls is
  unchanged.

- `internal/auth/source.go` + `auth/client.go` + `api/client.go` +
  `transport/client.go`: consolidated the `slog.New(slog.NewTextHandler(io.Discard, nil))`
  discard-logger pattern. Each of the three packages now has a single
  package-level `var discardLogger` built once at init; previously the
  same expression was inline-duplicated across constructors and
  `logger()` helpers (and `transport` had a `discardLogger()` function
  that re-allocated per call). `NewSessionTokenSource` now seeds
  `Logger` to that discard logger in its constructor, matching the
  pattern already used by `auth.New`, `api.New`, and `transport.New`.
- `internal/cli/output.go` + `cli/root.go`: `formatNames` is now a
  package-level `var` (built once at init) instead of a function that
  re-built the `[]string` on every call. `ParseFormat`'s error message
  and `joinFormats` consume it as a value; both call sites are
  read-only, so sharing the backing array is safe.
- `internal/cli/output.go`: tightened the `ErrTableNotSupported`
  message to a single phrase per Go error-style guidance — same
  meaning, but easier to skim in logs.
- `internal/soap/value.go`: `KindUnknown`'s doc now explains why the
  sentinel value is exactly 255 (max of `Kind`'s underlying `uint8`,
  giving the iota block room to grow without collision).
- `internal/cli/wire.go`: documented why the `auth.New(... soap.AuthPlain, authOpts)`
  call inside the `auth_type=session` branch hardcodes `AuthPlain` — KasAuth
  always bootstraps in plain mode regardless of session mode, and the
  session token is what subsequent KasApi calls use. Prevents future
  drive-by "fixes" that would replace the constant with `AuthSession`.
- `internal/transport/client.go`: documented why `waitGate` runs once
  before the retry loop, not inside it. 5xx-driven retries do not decode
  envelopes, so no fresh `KasFloodDelay` can be recorded between attempts
  — re-checking the gate would always be a no-op.
- `internal/auth/source.go`: `SessionTokenSource` now exposes a `Logger`
  field; the three previously silent `s.Store.Save` / `s.Store.Delete`
  call sites now log a `Warn` event when persistence fails so disk-full
  or permission issues surface in `--verbose` output. The in-memory
  cache still works in that case and the next invocation re-bootstraps
  via KasAuth, so behaviour is unchanged for the success path. The CLI
  wires `--verbose` into the new field via `internal/cli/wire.go`.
- `internal/cli/output.go` + `internal/cli/root.go`: deduplicated the
  `AllFormats → []string` conversion. A new package-internal
  `formatNames()` is now the single source of truth used by both the
  `--output` flag help (joined with `|`) and the `ParseFormat` error
  message (joined with `, `).
- `internal/soap/value.go`: replaced the magic `Kind(255)` sentinel
  returned by `classifyType` for unknown `xsi:type` values with a named
  `KindUnknown = 255` constant. Documented why the constant lives
  outside the iota block (so future enum additions cannot collide).
- `internal/cli/output.go`: clarified the `ErrTableNotSupported`
  message. Every subcommand result type is expected to implement
  `Tabular`, so this error always indicates a kasapi-cli bug rather than
  a user choice; the message now says so while keeping the
  `--output=json` / `--output=yaml` workaround hint.
- `internal/soap/envelope.go` + `internal/auth/codec.go`: wrap the SOAP
  decoder's input reader in `io.LimitReader(r, soap.MaxResponseBytes)`
  (16 MB) and set `dec.Strict = true` explicitly. The KAS server is
  trusted, so this is defense-in-depth against a malformed or hostile
  response (compromised endpoint, MITM, server bug) — the largest
  captured fixture is ~70 KB, so the cap is well above any legitimate
  payload.
- `internal/cli/wire.go`: rename `sessionOpts.any()` to `isSet()` to
  stop shadowing the Go 1.18 builtin alias `any` in IDE auto-complete
  and code review. Behaviour and the single call site are unchanged.
- `internal/session/store.go`: document why the explicit
  `tmp.Chmod(0o600)` after `os.CreateTemp` is kept — redundant on Unix
  but Windows ignores the create-mode bits, so the set is
  defense-in-depth across platforms.
- test: `internal/account` — added unit tests for `Client.Settings`,
  `Client.Resources`, plus the previously-unexercised `TableHeaders`
  on `AccountResources` and `AccountSettings` (and the `TableRows` of
  `AccountSettings`). Lifts package coverage from 69.9% to 85.4%.
- test: `internal/cli` — added help-output tests for the `dns`,
  `domains`, `subdomains`, `tlds`, and `mail` (incl. `accounts` /
  `forwards` / `filters` / `lists` subgroups) command factories,
  mirroring the pattern already used for the
  cronjobs/databases/... factories. Lifts package coverage from 60.0%
  to 69.9%.
- test: `internal/auth` — added negative-path and nil-input tests for
  the `IsLoginFailed`, `IsLoginLocked`, `IsUnknownSession`,
  `IsOTPPinIncorrect`, `IsCode`, and `AsError` sentinel helpers.

### Fixed

- Exit-code classification for `sessions delete` (re-review
  follow-up): a failure to remove the local `sessions.toml` entry now
  exits with the user-error code (1) instead of the API-error code
  (2). A local cache-removal failure is a client-side problem, not a
  KAS fault or network failure, so it now matches the same
  classification the `gen-docs` filesystem failures already use. The
  truthful "could NOT be cleared" message and the non-zero exit are
  unchanged; only the code differs.

- Monotonic flood-delay gate (review follow-up):
  `transport.Client.RecordDelay` now extends the gate to `now+d` only
  when that is later than an already-pending deadline, instead of
  unconditionally overwriting it. Previously a shorter `KasFloodDelay`
  arriving while a longer gate was still active reset the gate to the
  shorter window, which could let the client resume early and trip the
  server's flood protection. An explicit zero/negative delay still
  clears the gate unconditionally (unchanged contract).

- Read-path safety and transport cancellation context (review
  follow-up): the generic `kasread.ListGet.Get` accessor now returns an
  explicit `"<label>: %q matched N entries (expected unique)"` error
  when a singular-variant lookup comes back with more than one entry,
  instead of silently returning the first — enforcing the documented
  "single matching entry" contract for every module's `Get`. Transport
  flood-gate and retry-backoff sleeps that are interrupted by context
  cancellation now wrap the error with the phase
  (`flood-gate wait interrupted` / `retry backoff interrupted`) while
  preserving `errors.Is(err, context.Canceled)`. The `internal/ddns`
  field-availability comment was corrected to match the captured
  fixtures (both the list and singular variants return
  `dyndns_target_ipv4` / `ipv6`).

- Truthful CLI output and exit-code classification (review follow-up):
  `kasapi-cli sessions delete` and `config use-profile` no longer claim
  the local session cache was cleared (or that the server-side session
  was invalidated) when the underlying `store.Delete` / `delete_session`
  call actually failed — the message now reflects what really happened,
  and `sessions delete` returns a non-zero exit when the local cache
  removal fails instead of swallowing it. `gen-docs` local filesystem
  failures (`mkdir` / write) now map to the user-error exit code (1)
  via `UserError` instead of falling through as the API-fault code (2).
  No behaviour change to the success paths.

- `internal/session/store.go`: serialise `Load` / `Save` / `Delete`
  through an advisory file lock (`github.com/gofrs/flock`) at
  `<sessions.toml>.lock`. Previously two `kasapi-cli` processes running
  in parallel (scripts, CI pipelines) could race on the read-modify-
  write cycle so a Heartbeat from one silently lost a Save from
  another. Atomic temp+rename already protected the file against
  corruption; this fix protects the logical transaction. Worst-case
  symptom was a lost token causing one extra KasAuth refresh — the
  cache is self-healing, but surfacing the race is preferable.

- `.claude/skills/kasapi-cli-vertical-slice/SKILL.md`: corrected two
  factually wrong claims in the slice anatomy. The mapper naming
  convention is `Decode<Thing>` (e.g. `DecodeAccounts`), not
  `Map<Action>Response`; the client accessor lives on the **module's**
  own `*Client` in the module package (dispatching via a per-package
  `Caller` interface), not in `internal/api/client.go`. The transport-
  level `KasRequestParams` envelope is filled centrally in
  `internal/soap/request.go`; module code passes plain `map[string]any`
  to `Caller.Call`. `CONTRIBUTING.md:57-58` was already correct on both
  points; the skill drifted.
- `ROADMAP.md`: added a new `CLI write safety` section listing the
  cross-cutting prerequisites that gate every destructive subcommand —
  the destructive-write confirmation infrastructure (#109), the
  structured write-action audit log (#131), and `--dry-run` for write
  commands (#132). These are not KAS-API endpoints but block the entire
  v0.2.0 write phase; tracking them on the roadmap keeps the contributor
  view of "what is still pending" honest.
- `ROADMAP.md`: corrected the mail standard filter write entry from the
  non-existent `update_mailstandardfilter` to the actual KAS actions
  `add_mailstandardfilter` and `delete_mailstandardfilter` (the captured
  fixtures and issue #116 are authoritative; the KAS API has no
  `update_mailstandardfilter`).
- `CONTRIBUTING.md`: roadmap links no longer detour through
  `README.md#roadmap` (which is a one-line pointer with no checklist) —
  they now point directly at `ROADMAP.md`. The CI gate description was
  expanded to match what actually runs on every PR (`gosec`,
  `govulncheck`, `go build`, `docs sync`, `goreleaser config check`,
  CodeQL). The `kasapi-cli-vertical-slice` skill is now listed under
  authoritative references alongside `kasapi-cli-git-workflow` and
  `kasapi-cli-code-review`.
- `README.md`: dropped the incorrect "read and write operations" claim
  from the "What it does" paragraph (writes are still pending), tightened
  the Status sentence to mention the v0.1.0 read modules instead of
  "several", and corrected the output-format section (`table` is the
  default; the available formats are `json` / `yaml` / `table`, not just
  JSON).
- `docs/usage/mail.md`: corrected the description of `mail filters list`.
  The previous text claimed `get_mailstandardfilter` returns "server-side
  filter rules (Sieve-style: condition + action)"; the endpoint actually
  returns the catalog of pre-defined spam/virus filter presets that an
  account can attach via the `mail_spamfilter` setting on a mailaccount
  or forward.

## [0.1.0-alpha.1] - 2026-05-10

First public pre-release. Scope: the **read-only** KAS API surface
(accounts, server info, domains/subdomains/TLDs, DNS, mail, databases,
FTP/Samba users, cronjobs, directory protection, software installs,
DDNS users, usage statistics) plus session and plain authentication,
output formatters (`table` / `json` / `yaml`), config-file plumbing,
and the goreleaser pipeline (multi-arch binaries, deb/rpm,
keyless-cosign signatures, SPDX SBOMs). **Write endpoints are out of
scope** for this alpha and are tracked under v0.2.0 (#13). Expect
breaking changes between this alpha and the stable `v0.1.0` tag — CLI
flag names, output structures, and exit-code mappings are not yet
frozen.

### Fixed

- `.github/workflows/release.yml` cosign pin: `sigstore/cosign-installer@v3`
  was pinned to `v2.4.1`, which cannot read the bundle format that
  newer `goreleaser-action@v7` releases ship. The release workflow
  failed during `goreleaser-action`'s self-verification with
  `bundle does not contain cert for verification, please provide
  public key`. The pin is dropped so the installer picks its current
  default; goreleaser-action keeps in step with it.
- `.goreleaser.yaml` `archives.files` glob: `docs/cli/**/*` and
  `docs/usage/**/*` matched nothing (both directories are flat) so
  the rendered docs were missing from the release archives. Switched
  to `docs/cli/*` / `docs/usage/*` so the docs land alongside
  `LICENSE`, `README.md`, and `CHANGELOG.md` in every tarball/zip.
- `usage traffic --year` / `--month` range-validation errors now exit
  with code 1 (user error) instead of code 2 (API error). The
  `PreRunE` previously returned a plain `fmt.Errorf` which fell
  through `cli.CodeFor`'s default branch.
- `dns list --domain ""` now reports `required flag --domain not
  provided` without the redundant `--domain:` prefix and exits with
  code 1; the validation moved into a dedicated `PreRunE` so the
  body can use the canonical `runListE` shape.

### Changed

- `account.Client.List` now returns `account.AccountList` directly,
  matching every other read module. The CLI no longer needs to
  convert `[]Account` to the named list type, and the
  `kasread.ListGet` field is parameterised on `AccountList`.

- Collapsed the four remaining stand-alone `Caller interface`
  declarations in `internal/directoryprotection`, `internal/dns`,
  `internal/server`, and `internal/usage` to `type Caller =
  kasread.Caller`, matching the nine modules that were already
  aliased after issue #73's PR B. The shape was identical in all
  four cases; the alias removes the last bit of duplicated
  interface boilerplate and ensures a future change to the
  `Caller` contract is a single-file diff in `internal/kasread`.

### Documentation

- Closed the two remaining follow-ups from issue #73's NTH bundle:
  verified against the KAS docs that `get_mailstandardfilter`
  accepts no filter parameter (the `mailfilter.Client` doc-comment
  now links the spec page authoritatively, no code change needed),
  and added `TestClientGetNotFound` for `ddns.Client.Get` to pin
  the empty-array fallback that the prior test suite did not
  exercise.

### Changed

- Unified the `get`-subcommand `Use:` placeholders to the
  KAS-wire-parameter rule (filter key with hyphens):
  `<dyndns-login>` → `<ddns-login>`, `<subdomain>` →
  `<subdomain-name>`, `<address>` → `<mail-forward>`, `<domain>` →
  `<domain-name>`. The 8 placeholders that already followed the
  rule (e.g. `<ftp-login>`, `<cronjob-id>`, `<software-id>`) are
  unchanged. `docs/cli/` regenerated. Cosmetic finalisation of the
  cross-module-duplication clean-up bundle tracked in issue #73.
- Centralised the singular-record `[]string{"FIELD", "VALUE"}`
  table-header literal — duplicated 13 times across 11 read modules
  — into the new shared `internal/tablefmt` package's
  `FieldValueHeaders` variable. Each module's `TableHeaders()` for
  the singular view now returns the shared variable; a future
  rename ("KEY"/"VALUE", localisation, …) is one diff rather than
  thirteen. Part four of the cross-module-duplication clean-up
  bundle tracked in issue #73.
- Replaced the duplicated `cobra.RunE` bodies in 14 CLI files
  (account, cronjobs, databases, ddnsusers, directoryprotection,
  domains, ftpusers, mail, sambausers, server, softwareinstalls,
  subdomains, tlds, usage) with two generic factories
  `runListE[T]` / `runGetE[T]` in `internal/cli/run.go`. Each
  subcommand now passes a one-line closure that owns its module
  client construction and the actual call; the factory handles
  `BuildAPIClient`, `APIError(action)` wrapping, and `Render`. 28
  RunE bodies migrated; behaviour and exit codes are unchanged.
  Part three of the cross-module-duplication clean-up bundle
  tracked in issue #73.
- Replaced the duplicated `Client.List` / `Client.Get` boilerplate
  in 12 read modules (account, cronjob, database, ddns, domain,
  ftpuser, mailaccount, mailforward, mailinglist, sambauser,
  softwareinstall, subdomain) plus the `List`-only mailfilter with
  a single generic helper `kasread.ListGet[L, E]`. Each module now
  binds the action, label, filter key and decoder once in
  `NewClient` and exposes `List` / `Get` as one-line delegates;
  per-module `Caller` interfaces collapse to type aliases over
  `kasread.Caller`. Behaviour and error messages are unchanged.
  Part two of the cross-module-duplication clean-up bundle tracked
  in issue #73.
- Replaced the duplicated KAS `Array` of `Map` decoder boilerplate
  in 19 read decoders with a single generic helper
  `soap.DecodeArray[T]`. Each module's `Decode<Foo>s` now delegates
  the kind/item-shape checks to the helper and only declares its
  per-item mapper; behaviour and error messages are unchanged. Part
  one of the cross-module-duplication clean-up bundle tracked in
  issue #73.

### Added

- Vulnerability and security scanning, stage 2 of two: GitHub-native
  CodeQL workflow (`.github/workflows/codeql.yml`) running on PR +
  push to `main` + weekly cron, with the `security-extended` query
  pack; OSSF Scorecard workflow (`.github/workflows/scorecard.yml`)
  running weekly + on push to `main`, publishing the score for the
  public dashboard at <https://securityscorecards.dev>; `SECURITY.md`
  at the repo root with the disclosure policy, response expectations,
  and verification recipe (the short hint in `CONTRIBUTING.md` now
  links here); SBOMs (SPDX-JSON) generated per release artefact via
  goreleaser's `sboms:` block — Syft is now load-bearing in
  `release.yml`, so the previous `continue-on-error: true` was
  removed.

- Vulnerability and security scanning, stage 1 of two: a new
  `govulncheck` CI job (official Go vulnerability scanner, call-graph
  aware) on every PR + push to `main`; `gosec` added to the
  `golangci-lint` linter set in `.golangci.yml`; Dependabot configured
  for `gomod` and `github-actions` ecosystems via
  `.github/dependabot.yml`. Existing call sites that triggered gosec
  false-positives (test fixture loaders, `os.Stdin.Fd()` conversion,
  public-docs `MkdirAll` mode) were annotated with targeted
  `//nolint:gosec` markers carrying the rule ID and a one-line reason.
  CodeQL, OSSF Scorecard, `SECURITY.md`, and SBOM-in-release land in
  stage 2.

### Changed

- Go toolchain bumped from 1.23 to 1.25 (`go.mod` directive plus
  `go-version: "1.25"` in `ci.yml` and `release.yml`). Required to
  clear all reachable Go-stdlib CVEs that the new `govulncheck` job
  surfaced — 18 against 1.23, 7 of which (asn1 / net/url /
  encoding/pem / crypto/tls / crypto/x509 / os) only have 1.25.x
  backports, since 1.23 and 1.24 have both dropped out of the
  security-supported window. No source-level changes were needed
  for the bump itself.

- Release pipeline (`.goreleaser.yaml` + `.github/workflows/release.yml`)
  driven by `git tag v*`. Builds Linux + Windows × `amd64`/`arm64`
  binaries with `internal/version` ldflags wired up, packages Linux
  artefacts as `deb` and `rpm` via `nfpm`, ships tarballs (`tar.gz`) /
  ZIPs alongside, and signs every artefact plus `SHA256SUMS` keylessly
  with `cosign` via GitHub OIDC. A new CI job `goreleaser config check`
  validates `.goreleaser.yaml` on every PR. README gains an `Install`
  section pointing at the Releases page with a `cosign verify-blob`
  recipe. Makefile gets `release-snapshot` (local dry run into `./dist`,
  skips signing) and `release-check` targets. AUR `PKGBUILD` and a
  Homebrew tap are tracked separately as a follow-up.

- CI job `docs sync` (`.github/workflows/ci.yml`) that runs
  `make docs` and fails when the checked-in `docs/cli/` differs
  from the regenerated output. Ensures any change to a flag,
  subcommand registration, or short/long description comes paired
  with a `docs/cli/` refresh.

- Per-resource usage docs under `docs/usage/` (eight pages — `accounts`,
  `server`, `domains`, `dns`, `mail`, `databases`, `usage`, `hosting` —
  plus an index `README.md`). Each page lists the most common
  invocations, sketches the table / JSON output shape, and links to
  the matching KAS-API documentation page.

- Auto-generated Markdown CLI reference under `docs/cli/`, produced by
  a new hidden `kasapi-cli gen-docs <out-dir>` subcommand wrapping
  `cobra/doc.GenMarkdownTree`. The root command's `DisableAutoGenTag`
  is flipped before generation so re-running the generator produces
  byte-identical output when the CLI surface has not changed.

- Top-level `Makefile` with a `docs` target (`make docs`) that wipes
  `docs/cli/` and regenerates it via `go run ./cmd/kasapi-cli
  gen-docs docs/cli`. Other targets (`build`, `test`, `lint`, `vet`,
  `fmt`, `clean`) wrap the standard Go loop documented in
  `CONTRIBUTING.md`. Closes #35.

### Added

- `internal/ddns` read module and `kasapi-cli ddnsusers list|get`
  subcommand tree wrapping `get_ddnsusers`. The list variant decodes
  the Array of Maps into a typed `DDNSUserList`; `get <dyndns-login>`
  reuses the same endpoint with a `ddns_login` filter (note: the
  filter parameter has no `y`, unlike the response keys which use
  the `dyndns_*` prefix; per the KAS docs at `get-ddnsusers-inc.html`)
  and unwraps the single-entry result. The list view joins
  `dyndns_label` and `dyndns_zone` into a single `FQDN` column so
  the table reflects the hostname clients will actually look up;
  the explicit `dyndns_target_ipv4` / `dyndns_target_ipv6` fields
  surface as separate K/V rows in the singular view when the API
  populated them. `dyndns_password` is omitted from both table
  views (still available via `--output=json|yaml`). The KAS API
  signals "filter matched no entry" with a `dyndns_login_not_found`
  SOAP fault rather than an empty array; that fault propagates as
  an `*api.Error` and is detected by `api.IsNotFound`. Mapping tests
  run against `testdata/ddns/get_ddnsusers_response_success.xml` and
  `get_ddnsuser_response_success.xml`. Refs #11.

- `internal/softwareinstall` read module and `kasapi-cli softwareinstalls
  list|get` subcommand tree wrapping `get_softwareinstall` (note: the
  KAS action name is singular for both variants). The list variant
  decodes the Array of Maps into a typed `SoftwareInstallList`; `get
  <software-id>` reuses the same endpoint with a `software_id` filter
  and unwraps the single-entry result. The list view collapses the
  PHP and database `{from, upto}` version pairs into one column each
  ("8.4", "10.5..12.0"), prefixes the DB column with the engine name,
  and renders the `0.0` "not applicable" sentinel as `—`. The base64
  `image` data URI is kept on the struct for JSON/YAML round-trip
  fidelity but stripped from both table views. Mapping tests run
  against `testdata/softwareinstall/get_softwareinstalls_response_success.xml`
  (22 entries) and `get_softwareinstall_response_success.xml`. Refs #11.

- `internal/directoryprotection` read module and
  `kasapi-cli directoryprotection list [--path PATH]` subcommand
  wrapping `get_directoryprotection`. The KAS endpoint returns one
  entry per `(directory_path, directory_user)` tuple, so a directory
  with N users surfaces as N rows; for that reason this slice is
  exposed as a list with an optional `--path` filter rather than the
  usual list+get pair (matching the `dns list --domain` shape).
  `directory_password` is omitted from the table view but remains
  available via `--output=json|yaml`. Mapping tests run against
  `testdata/directoryprotection/get_directoryprotections_response_success.xml`
  and `get_directoryprotection_response_success.xml`. Refs #11.

- `internal/cronjob` read module and `kasapi-cli cronjobs list|get`
  subcommand tree wrapping `get_cronjobs`. The list variant decodes
  the Array of Maps into a typed `CronjobList`; `get <cronjob-id>`
  reuses the same endpoint with a `cronjob_id` filter and unwraps
  the single-entry result, matching the established read-slice
  pattern. The list view collapses the five schedule fields into a
  single crontab(5)-style `SCHEDULE` column and renders the trigger
  target as either `protocol://http_url` or `shell_command`; the
  singular view keeps the raw fields plus the joined schedule.
  `xsi:nil` values for `shell_command` / `timeout` round-trip cleanly
  to zero values flagged with `omitempty`. Mapping tests run against
  `testdata/cronjob/get_cronjobs_response_success.xml` and
  `get_cronjob_response_success.xml`. Refs #11.

- `internal/sambauser` read module and `kasapi-cli sambausers list|get`
  subcommand tree wrapping `get_sambausers`. The list variant decodes
  the Array of Maps into a typed `SambaUserList`; `get <samba-login>`
  reuses the same endpoint with a `samba_login` filter (per the KAS
  API docs at `get-sambausers-inc.html`) and unwraps the single-entry
  result, matching the mail accounts / accounts / databases pattern.
  The list view shows login, path, comment, and `in_progress`; the
  singular view falls back to a key/value table and omits
  `samba_password` (still available via `--output=json|yaml`).
  Mapping tests run against
  `testdata/sambauser/get_sambausers_response_success.xml` and
  `get_sambauser_response_success.xml`. Refs #11.

- `internal/ftpuser` read module and `kasapi-cli ftpusers list|get`
  subcommand tree wrapping `get_ftpusers`. The list variant decodes
  the Array of Maps into a typed `FTPUserList`; `get <ftp-login>`
  reuses the same endpoint with an `ftp_login` filter and unwraps the
  single-entry result. The list view shows login, path, comment,
  main-user flag, the three permission flags (R/W/L), the ClamAV
  scan flag, and `in_progress`; the singular view falls back to a
  key/value table and omits `ftp_password` / `ftp_passwort` (still
  available via `--output=json|yaml`). Mapping tests run against
  `testdata/ftpuser/get_ftpusers_response_success.xml`,
  `get_ftpuser_response_success.xml`, and the empty-list fixture
  (`get_ftpuser_response_success_empty_list.xml`). Refs #11.

- `internal/database` read module and `kasapi-cli databases list|get`
  subcommand tree wrapping `get_databases`. The list variant decodes
  the Array of Maps into a typed `DatabaseList`; `get <database-login>`
  reuses the same endpoint with a `database_login` filter and unwraps
  the single-entry result, mirroring the mail accounts / accounts
  pattern. The list view reports `used_database_space` in MB; the
  singular view uses a key/value table and omits `database_password`
  (still available via `--output=json|yaml`). Mapping tests run against
  `testdata/database/get_databases_response_success.xml` and
  `get_database_response_success.xml`. Refs #11.

- `account.Client.Get(ctx, login)` and `kasapi-cli accounts get
  <account-login>` calling `get_accounts` with an `account_login`
  filter. The result is unwrapped from the single-entry array so the
  CLI can render a key/value detail view; an empty array surfaces as a
  not-found error. Mapping test runs against
  `testdata/account/get_account_response_success.xml`.

- `internal/mailinglist` read module and `kasapi-cli mail lists list|get`
  subcommand tree wrapping `get_mailinglists`. The list variant decodes
  the Array of `{mailinglist_name, mailinglist_admin, mailinglist_url,
  in_progress}` Maps into a typed `MailingListList`; `get <name>` reuses
  the same endpoint with a `mailinglist_name` filter (per the KAS docs
  at `get-mailinglists-inc.html`) and unwraps the single-entry result,
  mirroring the mail-forwards pattern. The singular view falls back to
  a key/value table so the URL stays readable without truncation.
  Mapping tests run against
  `testdata/mailinglist/get_mailinglists_response_success.xml` and
  `get_mailinglist_response_success.xml`. Closes #9.

- `internal/mailfilter` read module and `kasapi-cli mail filters list`
  subcommand wrapping `get_mailstandardfilter`. Decodes the Array of
  `{filter, type, title, recommended}` Maps into a typed
  `StandardFilterList` so callers can resolve the preset filter ids
  used by `mail_spamfilter` on accounts/forwards. Mapping test runs
  against `testdata/mailfilter/get_mailstandardfilter_response_success.xml`.
  Refs #9.

- `internal/mailforward` read module and `kasapi-cli mail forwards
  list|get` subcommand tree wrapping `get_mailforwards`. The list
  variant decodes the full Map-of-Maps payload into a typed
  `MailForwardList`; `get <address>` reuses the same endpoint with a
  `mail_forward` filter (the source address) and unwraps the
  single-entry result, mirroring the mail accounts pattern. Mapping
  tests run against `testdata/mailforward/get_mailforwards_response_success.xml`
  and `get_mailforward_response_success.xml`. Refs #9.

- `internal/mailaccount` read module and `kasapi-cli mail accounts
  list|get` subcommand tree wrapping `get_mailaccounts`. The list
  variant decodes the full Map-of-Maps payload into a typed
  `MailAccountList`; `get <mail-login>` reuses the same endpoint with a
  `mail_login` filter and unwraps the single-entry result. The
  `--output=table` view shows login, address, used MB, responder flag
  and active state; the singular view falls back to a key/value table
  so the wider field set (xlist folders, 2FA flag, quota rule, webmail
  autologin) stays readable. Mapping tests run against
  `testdata/mailaccount/get_mailaccounts_response_success.xml` and
  `get_mailaccount_response_success.xml`. Refs #9.

- `kasapi-cli subdomains get <name>` calls `get_subdomains` with a
  `subdomain_name` filter and unwraps the single-entry result, mirroring
  the existing `domains get` flow; the singular `Subdomain` value
  renders as a key/value table with the SSL cert/key/CSR PEM bodies
  summarised as `<bytes,lines>`.

- `internal/domain`, `internal/subdomain`, and `internal/dns` read
  modules with the matching CLI subcommand trees: `kasapi-cli domains
  list` and `domains get <name>` (`get_domains`, the latter passing a
  `domain_name` filter and unwrapping the single-entry result),
  `kasapi-cli subdomains list` (`get_subdomains`), `kasapi-cli tlds
  list` (`get_topleveldomains`), and `kasapi-cli dns list --domain <d>
  [--nameserver <ns>]` (`get_dns_settings`). Domain types `Domain`,
  `SSL`, `TLD`, `Subdomain`, and DNS `Record` decode the KAS Map/Array
  payloads into typed Go values; the SSL cert/key/CSR PEM bodies are
  carried through but summarised as `<bytes,lines>` in the
  `--output=table` view of `domains get` so the key/value layout stays
  readable. Mapping tests run against the shipped `testdata/domain/`,
  `testdata/subdomain/`, and `testdata/dns/` fixtures. Closes #8.

- `internal/usage` package and `kasapi-cli usage` subcommand tree
  covering the three KAS read endpoints around webspace and traffic
  counters: `usage space` (`get_space`) lists per-account webspace
  totals with a usage ratio; `usage space-detail [--directory PATH]`
  (`get_space_usage`) reports per-directory file counts and byte sums;
  `usage traffic [--year Y --month M]` (`get_traffic`) returns the
  monthly summary plus per-day rows. The decoder maps the get_traffic
  Map keyed by `0` / `01..31` into a slice (summary first, then days),
  treats `xsi:nil` FTP fields as zero, and parses the xsd:string-encoded
  byte counts into `int64` so 9-digit values survive on 32-bit
  platforms. Closes #10.

- `internal/session` persistent session-token cache so a successful
  KasAuth login (including 2FA via `--otp`) survives across CLI
  invocations: a new `sessions.toml` next to the config file (mode
  `0600`, atomic temp+rename) stores `{token, expires_at,
  lifetime_seconds, update_lifetime}` keyed by login.
  `auth.SessionTokenSource` now loads the cached entry on first use,
  reuses it while `expires_at` has not been reached, persists every
  fresh KasAuth response, and deletes the entry on `Invalidate`.
  Lifetime defaults to `session.DefaultLifetime` (24 h, matching the
  KasAuth `session_lifetime` default) when `--session-lifetime` was
  not set; otherwise it mirrors the flag value. With
  `--session-update-lifetime Y`, `api.Client.Call` now invokes a new
  optional `Heartbeater` interface on the token source after every
  successful call so the local `expires_at` rolls forward in lockstep
  with the server-side window. Practical effect: rerun a command and
  no `--otp` prompt is needed for as long as the session is alive.

### Changed

- `CLAUDE.md`: rewrite stale Repository State paragraph that still
  described the project as greenfield (no `cmd/`, `internal/`,
  `go.mod`, no git repository, no build/test runnable). Replace
  with the current state: read-phase modules wired up, `main`
  protected with required signatures, CI gate (`lint & test` +
  `docs sync`) green on every push. Fix the `testdata/` filename
  convention to match the real layout (`<module>/<kas_action>_
  response_<status>[_<variant>].xml`, not `get_<thing>.xml`). Add
  pointers to the `kasapi-cli-git-workflow` /
  `kasapi-cli-code-review` skill files and `CONTRIBUTING.md`
  alongside the existing `docs/go/` references. Wire the standard
  command loop to the `Makefile` targets that exist today.

- `go.mod`: bump dependency pins after a routine audit pass —
  `golang.org/x/term` v0.30.0 → v0.34.0 (last release that still
  builds against the project's `go 1.23.0` baseline; v0.35.0+
  requires Go 1.24, v0.41.0+ requires Go 1.25 — out of scope for
  this loop). Indirect bumps: `golang.org/x/sys` v0.31.0 →
  v0.35.0, `github.com/spf13/pflag` v1.0.9 → v1.0.10,
  `github.com/cpuguy83/go-md2man/v2` v2.0.6 → v2.0.7. The
  generated `docs/cli/` is byte-identical after the bump.

### Fixed

- `docs/usage/`: replace 404 KAS-API anchor URL
  (`packages/API%20Functions.html`, used as a generic placeholder
  in every page) with per-function `files/<kas_action>-inc.html`
  URLs. Five referenced KAS actions did not exist — replace with
  the canonical names captured in `testdata/`: `get_accountusage`
  → `get_space`, `get_accountusagedetail` → `get_space_usage`,
  `get_accounttraffic` → `get_traffic`, `get_tlds` →
  `get_topleveldomains`, `get_mailfilter` →
  `get_mailstandardfilter`. All 24 external doc links in the
  user-facing markdown set now resolve to HTTP 200.

- `--verbose` / `-v` was bound to a `RootOptions` field but never
  read anywhere; the flag was effectively a no-op. Plumb a
  `*slog.Logger` (text handler on stderr when verbose, discard
  otherwise) through `BuildAPIClient` into `transport.Client`,
  `api.Client`, and `auth.Client`. Events emitted: resolved
  credentials with `auth_data` redacted (`cli`); SOAP action
  before each request, auth-failure retry, `flood_protection`
  fallback gate, applied `KasFloodDelay` (`api`); `KasFloodDelay`
  gate wait, transient-error retry attempt (`transport`);
  `KasAuth` bootstrap and credential-token issued with login +
  token length only (`auth`). Stdout stays clean for `-o json |
  jq` pipes; logs go to stderr only.

### Changed

- `README.md`: expand with a Configuration section (TOML profile
  example for both `auth_type=plain` and `auth_type=session`,
  `KAS_LOGIN`/`KAS_AUTHDATA`/`KAS_AUTHTYPE` env-var reference,
  flag/env/profile precedence), a Quick start section with the
  read commands that exist today (`accounts list|get|resources`,
  `server info`, `--output` formats), and a Troubleshooting section
  covering `KasFloodDelay`, the `no_auth`/`unknown_session`/
  `kas_session_invalid` retry behaviour, `--verbose`, and a pointer
  at the signed-commit / branch-protection rules in `CONTRIBUTING.md`.
  Closes #14.

- `CONTRIBUTING.md`: add explicit pointers to the
  `kasapi-cli-git-workflow` and `kasapi-cli-code-review` skill
  files alongside the existing references to `AGENTS.md` and the
  `docs/go/` set; absorb the contributor-facing "Repository layout"
  section that previously lived in `README.md`.

- `internal/testutil`: extract the `repoRoot` / `decodeFixture` /
  `fakeCaller` helpers that every per-module `*_test.go` carried as
  a private copy into a single shared package, and migrate all 20
  test files (17 module test suites + `internal/{soap,auth,api}`'s
  own root-discovery helpers) to use it. `DecodeFixture` now takes a
  forward-slash-separated path rooted at `testdata/` (e.g.
  `"mailinglist/get_mailinglists_response_success.xml"`) so the fixture
  layout is visible at the call site instead of being hidden inside
  per-module `decodeFixture(t, name)` wrappers. The `FakeCaller` stub
  is exported with `Resp` / `Err` / `GotAction` / `GotParams` fields
  so cross-module test code can construct it directly. Net effect:
  ~910 lines of word-for-word boilerplate removed; behaviour
  unchanged (the loop ends only after `go test -race ./...` and
  `golangci-lint run` are clean against the migrated suite).

- `kasapi-cli mail lists get` argument placeholder renamed from
  `<name>` to `<mailinglist-name>` so the help text matches the
  KAS wire parameter and the placeholder convention used by every
  other `get` subcommand (`<address>`, `<mail-login>`, `<dyndns-login>`,
  `<domain>`, `<software-id>`, …). `TestMailingListSingularTabular`
  was tightened from a map-lookup over rows (order-insensitive) to
  an indexed comparison so a future refactor reordering `TableRows`
  cannot slip past the test silently.

- `internal/soap`: extend `Value` with typed Map accessors (`MapString`,
  `MapInt`, `MapInt64`, `MapFloat`) and a generic `AsInt` coercion so
  every read module can drop its private `getString` / `getInt` /
  `getInt64` / `getFloat` helper. Migrated 17 read packages to the new
  accessors (~28 helper copies removed; net –195 lines). Behaviour is
  unchanged — the new methods replicate the existing nil-safe coercion
  rules and are pinned by `TestValueMapAccessors` in
  `internal/soap/soap_test.go` against missing-key, cross-kind, and
  unparseable inputs. The package-local `getBool` / `getYN` in
  `internal/account/decode.go` are intentionally left in place; they
  are only used by one decoder and cover boolean/Y-N coercion that is
  out of scope for #56. Closes #56.

- `kasapi-cli accounts get` was renamed to `kasapi-cli accounts
  settings`; the old name now wraps `get_accounts` with the
  `account_login` filter (see Added), matching the `mail accounts
  list|get` pattern. The `accounts list` short description was
  tightened to clarify that an unfiltered `get_accounts` returns every
  account visible to the login (every sub-account for a main login,
  just the login itself for a sub-account).

### Changed

- `internal/usage`: add a `(t Traffic) IsSummary() bool` helper so
  callers no longer rely on the `Day == 0` magic number to distinguish
  the monthly summary row from per-day entries; the table renderer is
  switched over too. Document on `Space` that `UsedWebspace` is the sum
  of the four sub-buckets so future readers do not double-count.

- `kasapi-cli usage traffic`: pre-validate `--year` (must be in
  `[2000, currentYear+1]`) and `--month` (must be `1..12`) instead of
  forwarding obvious typos to KAS. Closes #45.

### Changed

- `internal/api/doc.go`: stop enumerating the auth-failure code list
  inline; point to `IsAuthFailure` as the single source of truth so a
  future code addition only has to update one place.

- `internal/auth/source.go`: extend the `SessionTokenSource`
  type-level doc to describe the snapshot/restore semantics applied
  during `Invalidate`, so readers see the full lifecycle without
  having to drill into the field block.

- `internal/api`: add `testdata/response_failed_kas_session_invalid.xml`
  and `TestCallRetriesOnSessionInvalid` to pin the full Client + retry
  composition for the new code; complements the IsAuthFailure
  table-test row.

### Fixed

- Session re-authentication now triggers on `kas_session_invalid`.
  `IsAuthFailure` previously covered only `no_auth`, `unknown_session`,
  `kas_access_forbidden`, and `got_no_login_data`; KAS also returns
  `kas_session_invalid` when a server-side session is no longer
  accepted (e.g. it was created with `session_update_lifetime=N` and
  the lifetime elapsed). Without this code the auto-retry path in
  `*api.Client.Call` did not fire and the user saw the raw fault.

- `internal/auth/source.go`: preserve the user-configured `Lifetime` /
  `UpdateLifetime` across `Invalidate`. When a persisted session was
  loaded its server-side properties are adopted for the duration of
  that session's life (so Heartbeat stays consistent), but the wired
  CLI-flag values are now snapshotted on first `Credentials` call and
  restored by `Invalidate`. The fresh session created by the next
  re-authentication therefore reflects the current run's flags rather
  than the stale persisted properties — fixing the case where an
  initial run without `--session-update-lifetime` would otherwise
  pin the persisted entry to `update_lifetime=false` forever.

- `internal/auth/source.go`: sharpen the `Heartbeat` doc comment.
  The previous wording claimed Heartbeat was a no-op "when no Store
  is wired up", but the in-memory rolling window is updated regardless
  of `Store`; the comment now describes the actual conditions
  (`UpdateLifetime` false or no cached token). Closes #41.

- `internal/api/client.go`: drop the stale "(issue #5)" reference from
  the `StaticTokenSource` doc comment. Issue #5 was closed by PR #29
  when the KasAuth client landed; the surrounding sentence is kept.

- `internal/account/table.go`: document the `used_account_space`
  unit conversion. The KAS phpdoc does not state the unit, but the
  magnitudes and fractional digits in real responses are consistent
  with KiB (`bytes/1024`); a one-line code comment records the
  derivation so future readers do not rediscover it.

- `internal/usage`: drop the action name from `DecodeSpace` /
  `DecodeSpaceUsage` / `DecodeTraffic` error strings. The Client
  wrappers already prepend `usage: get_space:` / `usage: get_traffic:`
  etc., so leaving the action in the decoder produced a doubled prefix
  (`usage: get_space: usage: get_space: ReturnInfo[0] is not a Map`).
  Decoders now use `"usage: …"` only, matching the established
  `account` / `server` pattern.

- `kasapi-cli config init`: rename the local `--profile` flag to
  `--name` so it no longer shadows the persistent root `--profile`
  flag. Previously `kasapi-cli --profile X config init` silently
  reverted to the local default `main` instead of writing profile `X`.
  The persistent `--profile` flag continues to select which profile is
  *used* at runtime; `--name` selects which profile is *written*.
- Replace the legacy direct `err == io.EOF` / `err != io.EOF`
  comparisons in `internal/cli/confirm.go` and `internal/auth/codec.go`
  with `errors.Is(err, io.EOF)` so wrapped EOF values are still
  recognised.

- `internal/transport`: drop the manual `Accept-Encoding: gzip`
  request header. `net/http` only decompresses gzip responses
  transparently when the caller has *not* set that header; the manual
  set turned automatic decoding off and leaked raw gzip bytes into the
  XML decoder, surfacing as `XML syntax error: invalid character
  entity &…` on the first kasserver response that came back compressed.
  Removing the header lets `net/http` add it (and decode the response)
  itself.

### Added

- `kasapi-cli config` subcommand tree for first-run bootstrap and
  inspection without hand-writing TOML: `config init` interactively
  prompts for `login`, `auth_type` (`session`|`plain`, defaulting to
  `session`), and `auth_data` (hidden via `golang.org/x/term.ReadPassword`),
  writes the profile to the resolved config path with mode `0600`
  (parent dirs created `0700`, atomic temp+rename), refuses to
  overwrite an existing profile unless `--force`, and offers to set
  `default_profile` when none is configured. `--profile` selects the
  profile name (default `main`). Non-TTY stdin fails fast with a clear
  error so CI and pipes do not hang. `config show` prints the resolved
  effective configuration after the flag/env/profile merge with
  `auth_data` redacted via `Credentials.String`. `config path` prints
  the resolved config-file path. `config.Save` is the new persistence
  helper that backs `config init`. (Closes #34.)
- `--otp`, `--session-lifetime`, and `--session-update-lifetime`
  persistent flags on the root command, exposing the optional KasAuth
  parameters (`session_2fa`, `session_lifetime`, and
  `session_update_lifetime`). All three are plumbed through
  `BuildAPIClient` into `auth.Options` so `auth.SessionTokenSource`
  forwards them on the credential-token bootstrap. `--session-lifetime`
  is range-checked client-side (1..30000 seconds);
  `--session-update-lifetime` accepts `Y` or `N` and maps to the tri-state
  `*bool` field. The `--auth-type` help text spells out that these flags
  are KasAuth-only (the KAS docs do not cover them on direct
  `kas_auth_type=plain` calls), so combining any of them with
  `auth_type=plain` is rejected up front with a user-error exit code and
  a message that points to `auth_type=session`.
- `internal/account` and `internal/server` read modules with the first
  end-to-end CLI subcommands: `kasapi-cli accounts list` (`get_accounts`),
  `kasapi-cli accounts get` (`get_accountsettings`), `kasapi-cli accounts
  resources` (`get_accountresources`), and `kasapi-cli server info`
  (`get_server_information`). Domain types `Account`, `AccountSettings`
  (with SSH fingerprints, user_prefs, direct-link flags), `AccountResources`
  / `ResourceQuota`, and `Service` / `ServiceList` decode the KAS Map/Array
  payloads into typed Go values; `ResourceQuota.Max == -1` is rendered as
  `∞` in the table view to match the documented "unlimited" sentinel.
  Mapping tests run against the shipped `testdata/account/get_*_response_success.xml`
  fixtures. `cli.BuildAPIClient(opts)` is the new wiring helper that reads
  config + env + flags, picks `api.StaticTokenSource` for `auth_type=plain`
  and `auth.SessionTokenSource` for `auth_type=session`, and returns an
  `*api.Client` that subcommands consume. (Closes #7.)
- `internal/cli` CLI scaffold built on [spf13/cobra](https://github.com/spf13/cobra):
  `NewRootCmd()` returns the `kasapi-cli` root command with persistent
  global flags `--config`, `--profile`, `--login`, `--auth-data`,
  `--auth-type`, `--output`, `--no-color`, `--verbose`, `--yes`, plus the
  built-in `--help` and `--version`. Output renderers (`json`, `yaml`,
  `table`) live behind a single `Render(w, format, v)` entry point;
  `--output=table` requires the value to implement the `Tabular`
  interface. A `Confirm(in, out, prompt)` helper covers the `[y/N]`
  prompt for future destructive write commands. `ExitError`,
  `UserError(...)`, `APIError(...)`, and `CodeFor(err)` translate failures
  to the documented exit codes (`0` ok, `1` user error, `2` API error);
  flag-parsing errors are routed through `UserError`. The binary is
  intentionally without subcommands until #7 — `kasapi-cli` prints help
  and `kasapi-cli --version` prints the build banner. Adds the
  `goccy/go-yaml` (active fork replacing the archived `gopkg.in/yaml.v3`)
  and `spf13/cobra` dependencies. (Closes #12.)
- `internal/auth` KasAuth.php credential-token client: separate codec
  (`tns:KasAuth` envelope, bare `xsd:string` token in `<return>`),
  `Client.GetCredentialToken(ctx)` returning the 40-character token,
  `Options{Lifetime, UpdateLifetime, OTP}` for the optional
  `session_lifetime`, `session_update_lifetime`, and 2FA
  `session_2fa` parameters. Faults surface as typed `*Error` with
  helpers `IsLoginFailed`, `IsLoginLocked`, `IsOTPPinIncorrect`,
  `IsUnknownSession`. `SessionTokenSource` adapts the client to the
  `api.TokenSource` interface, caching the token and re-fetching on
  `Invalidate` so `api.Client` can refresh transparently after an
  auth failure. (Closes #5.)
- `internal/api` generic KasApi.php call surface composing the soap codec
  with the http transport: `Client.Call(ctx, action, params)` encodes,
  posts, decodes, and feeds the server-reported `KasFloodDelay` back to
  the transport gate. SOAP-ENV:Fault bodies surface as typed `*Error`
  values whose `Code` is the stable KAS error string, with predicates
  (`IsAuthFailure`, `IsFloodProtection`, `IsNotFound`, `IsSyntaxError`,
  `IsMaxReached`, `IsInProgress`, `IsMissingParameter`, `IsNothingToDo`).
  A `TokenSource` interface plus `StaticTokenSource` provide credentials;
  `no_auth` and `unknown_session` trigger one token refresh and retry.
  (Closes #6.)
- `internal/transport` HTTP client wrapping the KAS SOAP endpoints:
  POST with the SOAP 1.1 content type, version-stamped User-Agent,
  exponential backoff on 5xx and network errors (4xx and SOAP faults
  are returned without retry), context-aware cancellation, and a
  per-client `RecordDelay`/gate pair so callers can honour the
  server-side `KasFloodDelay`. `Now`/`Sleep` are injectable for
  deterministic tests via `httptest.Server`. (Closes #4.)
- `internal/config` profile-aware credentials loader: TOML config under
  the OS-specific user-config path (XDG on Linux), multi-profile, with
  resolution precedence flag > env > profile > default profile. Env
  fallback via `KAS_LOGIN`, `KAS_AUTHDATA`, `KAS_AUTHTYPE`. Auth-data
  is redacted by `Credentials.String` so secrets do not surface in
  logs or `--help`. Validates `auth_type` (`plain` or `session`) and
  reports missing required fields. (Closes #2.)
- `internal/soap` codec for the KAS-API envelope: `Value` discriminated
  union mirroring the Apache xml-soap `ns2:Map` shape (xsi:type:
  string/int/float/boolean, ns2:Map, SOAP-ENC:Array), `Decode` for
  `KasApiResponse`/`SOAP-ENV:Fault` envelopes returning `*Response` or
  `*FaultError`, and `EncodeRequest` for the JSON-in-`<Params>` request
  envelope. Table-driven tests cover 471 response fixtures plus shape
  pins and encoder validation. (`testdata/session/` is left for the
  KasAuth client in issue #5.)
- Bootstrap Go module `github.com/chmmou/kasapi-cli` (Go 1.23).
- `cmd/kasapi-cli` entry point with build-stamped `--version`.
- `internal/` package skeleton mirroring the clean-architecture layering in
  `docs/go/ARCHITECTURE.md`: per-resource domain packages
  (`account`, `server`, `domain`, `subdomain`, `dns`, `mailaccount`,
  `mailforward`, `mailfilter`, `mailinglist`, `database`, `ftpuser`,
  `sambauser`, `cronjob`, `ddns`, `directoryprotection`, `softwareinstall`,
  `ssl`, `usage`, `chown`, `symlink`, `session`) plus inner-/adapter-layer
  packages (`soap`, `transport`, `auth`, `api`, `config`, `cli`, `version`).
- `.golangci.yml` matching the gate set in `docs/go/LINTING.md`.
- GitHub Actions CI workflow running `gofmt`, `go vet`, `golangci-lint`,
  `go test`, `go test -race`, and `go build ./cmd/kasapi-cli`.
