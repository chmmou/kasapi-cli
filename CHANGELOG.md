# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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

### Fixed

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
