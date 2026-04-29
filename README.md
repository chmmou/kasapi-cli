# kasctl

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

Early development. No functional Go code yet — the repository currently holds documentation, agent guidance, and recorded KAS-API response fixtures used to drive offline parser tests. See the project board for the active roadmap.

## What it does (planned)

`kasctl` is a command-line client for the All-Inkl KAS-API. It wraps the SOAP/`ns2:Map` wire format the API uses, handles the `KasAuth` credential-token flow (plain or session, optional 2FA), enforces the `KasFloodDelay` between calls, and exposes read and write operations for the resources documented at <https://kasapi.kasserver.com/dokumentation/phpdoc/>:

- accounts, account settings, account resources
- server information, space, space usage, traffic
- top-level domains, domains, subdomains, DNS settings
- mail accounts, mail forwards, mail standard filters, mailing lists
- databases, FTP users, Samba users, DDNS users
- cronjobs, directory protection, software install entries
- sessions (`add_session`, `delete_session`)

## Endpoints

- API: <https://kasapi.kasserver.com/soap/KasApi.php>
- Auth: <https://kasapi.kasserver.com/soap/KasAuth.php>

## Configuration (planned)

`kasctl` reads credentials from a config file or from environment variables (`KAS_LOGIN`, `KAS_AUTHDATA`, `KAS_AUTHTYPE`). Profiles let you switch between accounts. Secrets never appear in `--help` or in default log output.

## Building

Once the Go module is bootstrapped:

```sh
go build ./cmd/kasctl
go test ./...
```

## Repository layout

- `cmd/kasctl/` — CLI entry point (planned).
- `internal/` — domain types, transport, mappers (planned).
- `testdata/` — recorded KAS-API SOAP responses; the source of truth for response shape, used by offline parser tests.
- `docs/go/` — Go style, architecture, patterns, and linting reference for this repo.
- `.claude/skills/kasctl-git-workflow/` — git/PR workflow skill enforced for this project.
- `AGENTS.md`, `CLAUDE.md` — agent guidance.

## Contributing

Read `AGENTS.md` and `docs/go/ARCHITECTURE.md` before opening a PR. The git workflow (branch naming, commit style, PR shape, CI gate) is captured in `.claude/skills/kasctl-git-workflow/SKILL.md`.

## License

BSD 3-Clause — see [LICENSE](LICENSE).
