# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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
