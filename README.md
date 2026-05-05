<p align="center">
  <img src="assets/logo.png" alt="kasapi-cli logo" width="180">
</p>

# kasapi-cli

> [!IMPORTANT]
> An independent open-source KAS-API CLI written in Go that communicates with the KAS-API from All-Inkl.com. Not affiliated with All-Inkl.com.

## Disclaimer

> [!IMPORTANT]
> This tool interacts with the KAS API and can modify domains, DNS records, and other account-related settings.
>
> Use this software at your own risk.
>
> The author assumes no liability for any damage, data loss, service disruption, or misconfiguration caused by the use of this tool, to the extent permitted by applicable law.
>
> Always verify commands and test changes in a safe environment before applying them to production systems.
>
> This project is not affiliated with or endorsed by All-Inkl.com.

## Status

Early development. The transport, authentication, configuration, and several read modules are wired up; many write paths and the remaining read endpoints are still pending — see [ROADMAP.md](ROADMAP.md) for the current state. The repository also ships recorded KAS-API response fixtures under `testdata/` that drive offline parser tests.

## What it does

`kasapi-cli` is a command-line client for the All-Inkl KAS-API. It wraps the SOAP/`ns2:Map` wire format the API uses, handles the `KasAuth` credential-token flow (plain or session, optional 2FA), enforces the `KasFloodDelay` between calls, and exposes read and write operations for the resources documented at <https://kasapi.kasserver.com/dokumentation/phpdoc/>.

## Building

```sh
go build ./cmd/kasapi-cli
go test ./...
```

## Repository layout

- `cmd/kasapi-cli/` — CLI entry point.
- `internal/` — domain types, transport, mappers, CLI wiring; one package per KAS resource (see `internal/account/`, `internal/domain/`, `internal/dns/`, …) plus shared infrastructure (`internal/soap`, `internal/api`, `internal/auth`, `internal/transport`, `internal/session`, `internal/config`, `internal/cli`).
- `testdata/` — recorded KAS-API SOAP responses; the source of truth for response shape, used by offline parser tests.
- `docs/go/` — Go style, architecture, patterns, and linting reference for this repo.
- `.claude/skills/kasapi-cli-git-workflow/` — git/PR/merge mechanics enforced for this project.
- `.claude/skills/kasapi-cli-code-review/` — code-review loop (Blocker/Should/Nice classification, re-review cycle).
- `AGENTS.md`, `CLAUDE.md` — agent guidance.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, coding conventions, the vertical-slice pattern used per KAS endpoint, the commit/PR workflow, and the code-review loop.

## Roadmap

The current state of the KAS-API surface — implemented vs. pending — is tracked in [ROADMAP.md](ROADMAP.md).

## License

BSD 3-Clause — see [LICENSE](LICENSE).
