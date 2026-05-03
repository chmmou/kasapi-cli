# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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
