# kasapi-cli usage

Per-resource usage examples for the read surface that ships today. Each page lists the most common invocations, the shape of their output, and the matching KAS-API documentation page so you can correlate the response columns with the upstream contract.

For the full, auto-generated flag and subcommand reference (one Markdown file per command, regenerated from the live command tree via `make docs`), see [`../cli/`](../cli/).

For installation, configuration (TOML profiles, env vars, flag precedence), and troubleshooting (`KasFloodDelay`, `no_auth` / `unknown_session` recovery), see the [project README](../../README.md).

## Pages

- [accounts](accounts.md) — accounts, settings, resource quotas
- [server](server.md) — server-side hosting / service info
- [domains](domains.md) — domains, subdomains, registrable TLDs
- [dns](dns.md) — DNS records for a zone
- [mail](mail.md) — mail accounts, forwards, filters, mailing lists
- [databases](databases.md) — MariaDB / MySQL databases
- [usage](usage.md) — webspace and traffic counters
- [hosting](hosting.md) — FTP / Samba users, cronjobs, directory protection, software installs, DDNS users

## Conventions used in the examples

- `-o json` / `-o yaml` / `-o table` selects the output format. Default is `table`.
- Login identifiers in real KAS responses look like `w0000000`; passwords and tokens are always redacted in the examples below.
- All commands honour the global flags documented in [`../cli/kasapi-cli.md`](../cli/kasapi-cli.md).
