# Roadmap

This file tracks the KAS-API surface implemented by `kasapi-cli`, grouped by domain. Checked items are wired up end-to-end (typed module + CLI subcommand + fixture-backed tests); unchecked items are still pending.

The list is kept in sync with the code on `main`. To claim an unchecked item, please open an issue first so work is not duplicated — see [CONTRIBUTING.md](CONTRIBUTING.md).

## Transport & authentication

- [x] SOAP / `ns2:Map` decoder (`internal/soap`)
- [x] `KasFloodDelay` enforcement and retry on `flood_delay` errors
- [x] `KasAuth` credential-token flow (plain + session, optional 2FA via `--otp`)
- [x] Persistent session-token cache (`sessions.toml`) survives across CLI invocations
- [x] Session lifetime extension (`update_lifetime`)
- [x] Standalone `delete_session` subcommand (`kasapi-cli sessions delete`); `add_session` is the KasAuth credential-token flow already covered above (`internal/auth`), not a separate endpoint

## Configuration

- [x] TOML config file with named profiles and `default_profile` selection
- [x] Environment overrides (`KAS_LOGIN`, `KAS_AUTHDATA`, `KAS_AUTHTYPE`, `KAS_PROFILE`)
- [x] Secrets are never echoed in `--help` or default log output

## CLI write safety

Cross-cutting prerequisites for the v0.2.0 write phase — these are not KAS-API endpoints themselves but gate every destructive subcommand. All three are in place and wired by every landed write slice; the shared post-call seam lives in `internal/kaswrite`. The umbrella tracker for the write phase is [#13](https://github.com/chmmou/kasapi-cli/issues/13).

- [x] Destructive-write confirmation infrastructure (`--yes` / interactive `[y/N]` prompt, #109)
- [x] Structured write-action audit log (`--audit-log <path>` / `KAS_AUDIT_LOG`, #131)
- [x] `--dry-run` for write commands (preview KAS action + parameters without dispatching, #132)

## Accounts & server

- [x] `accounts list` / `accounts get <account-login>` (`get_accounts`, with `account_login` filter)
- [x] `accounts settings` / `accounts resources` (`get_accountsettings`, `get_accountresources`)
- [ ] Account write paths (`add_account`, `update_account`, `delete_account`, `update_accountsettings`, `update_superusersettings`, #110)
- [x] `server get` (`get_server_information`)

## Usage

- [x] `usage space` (`get_space`)
- [x] `usage space-detail` (`get_space_usage`)
- [x] `usage traffic` (`get_traffic`)

## Domains, subdomains, DNS

- [x] `domains list` / `domains get <name>` (`get_domains`, with `domain_name` filter)
- [x] `subdomains list` / `subdomains get <name>` (`get_subdomains`, with `subdomain_name` filter)
- [x] `tlds list` (`get_topleveldomains`)
- [x] `dns list --domain <d> [--record-id <id>]` (`get_dns_settings`)
- [ ] DNS write paths (`add_dns_settings`, `update_dns_settings`, `delete_dns_settings`, `reset_dns_settings`, #113)
- [ ] Domain write paths (`add_domain`, `update_domain`, `delete_domain`, `move_domain`, #111)
- [ ] Subdomain write paths (`add_subdomain`, `update_subdomain`, `move_subdomain`, `delete_subdomain`, #112)

## Mail

- [x] `mail accounts list` / `mail accounts get <mail-login>` (`get_mailaccounts`, with `mail_login` filter)
- [x] `mail accounts add/update/delete` (`add_mailaccount`, `update_mailaccount`, `delete_mailaccount`, #114)
- [x] `mail forwards list` / `mail forwards get <address>` (`get_mailforwards`, with `mail_forward` filter)
- [x] `mail forwards add/update/delete` (`add_mailforward`, `update_mailforward`, `delete_mailforward`, #115)
- [x] `mail filters list` (`get_mailstandardfilter`)
- [x] `mail filters add/delete` (`add_mailstandardfilter`, `delete_mailstandardfilter`, #116)
- [x] `mail lists list` (`get_mailinglists`)
- [x] `mail lists add/update/delete` (`add_mailinglist`, `update_mailinglist`, `delete_mailinglist`, #117)

## Hosting resources

- [x] `databases list` / `databases get <database-login>` (`get_databases`, with `database_login` filter)
- [x] `databases add/update/delete` (`add_database`, `update_database`, `delete_database`, #122)
- [x] `ftpusers list` / `ftpusers get <ftp-login>` (`get_ftpusers`, with `ftp_login` filter)
- [x] `ftpusers add/update/delete` (`add_ftpuser`, `update_ftpuser`, `delete_ftpuser`, #119)
- [x] `sambausers list` / `sambausers get <samba-login>` (`get_sambausers`, with `samba_login` filter)
- [x] `sambausers add/update/delete` (`add_sambauser`, `update_sambauser`, `delete_sambauser`, #120)
- [x] `ddnsusers list` / `ddnsusers get <dyndns-login>` (`get_ddnsusers`, with `ddns_login` filter)
- [x] `ddnsusers add/update/delete` (`add_ddnsuser`, `update_ddnsuser`, `delete_ddnsuser`, #121)
- [x] `cronjobs list` / `cronjobs get <cronjob-id>` (`get_cronjobs`, with `cronjob_id` filter)
- [x] `cronjobs add/update/delete` (`add_cronjob`, `update_cronjob`, `delete_cronjob`, #118)
- [x] `directoryprotection list [--path PATH]` (`get_directoryprotection`, optional `directory_path` filter)
- [x] `directoryprotection add/update/delete` (`add_directoryprotection`, `update_directoryprotection`, `delete_directoryprotection`, #123)
- [x] `softwareinstalls list` / `softwareinstalls get <software-id>` (`get_softwareinstall`, with `software_id` filter)
- [ ] Software install write path (`add_softwareinstall`, #124 — the KAS API exposes only the `add` action, no `update`/`delete`)
- [ ] Filesystem & SSL helpers (`add_symlink`, `update_chown`, `update_ssl`, #125)
