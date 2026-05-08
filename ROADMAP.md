# Roadmap

This file tracks the KAS-API surface implemented by `kasapi-cli`, grouped by domain. Checked items are wired up end-to-end (typed module + CLI subcommand + fixture-backed tests); unchecked items are still pending.

The list is kept in sync with the code on `main`. To claim an unchecked item, please open an issue first so work is not duplicated — see [CONTRIBUTING.md](CONTRIBUTING.md).

## Transport & authentication

- [x] SOAP / `ns2:Map` decoder (`internal/soap`)
- [x] `KasFloodDelay` enforcement and retry on `flood_delay` errors
- [x] `KasAuth` credential-token flow (plain + session, optional 2FA via `--otp`)
- [x] Persistent session-token cache (`sessions.toml`) survives across CLI invocations
- [x] Session lifetime extension (`update_lifetime`)
- [ ] Standalone `add_session` / `delete_session` subcommands

## Configuration

- [x] TOML config file with named profiles and `default_profile` selection
- [x] Environment overrides (`KAS_LOGIN`, `KAS_AUTHDATA`, `KAS_AUTHTYPE`, `KAS_PROFILE`)
- [x] Secrets are never echoed in `--help` or default log output

## Accounts & server

- [x] `accounts list` / `accounts get <account-login>` (`get_accounts`, with `account_login` filter)
- [x] `accounts settings` / `accounts resources` (`get_accountsettings`, `get_accountresources`)
- [x] `server get` (`get_server_information`)

## Usage

- [x] `usage space` (`get_space`)
- [x] `usage space-detail` (`get_space_usage`)
- [x] `usage traffic` (`get_traffic`)

## Domains, subdomains, DNS

- [x] `domains list` / `domains get <name>` (`get_domains`, with `domain_name` filter)
- [x] `subdomains list` / `subdomains get <name>` (`get_subdomains`, with `subdomain_name` filter)
- [x] `tlds list` (`get_topleveldomains`)
- [x] `dns list --domain <d> [--nameserver <ns>]` (`get_dns_settings`)
- [ ] DNS write paths (`add_dns_settings`, `update_dns_settings`, `delete_dns_settings`)
- [ ] Domain write paths (`add_domain`, `update_domain`, `delete_domain`, transfer flow)
- [ ] Subdomain write paths (`add_subdomain`, `update_subdomain`, `move_subdomain`, `delete_subdomain`)

## Mail

- [x] `mail accounts list` / `mail accounts get <mail-login>` (`get_mailaccounts`, with `mail_login` filter)
- [ ] Mail account write paths (`add_mailaccount`, `update_mailaccount`, `delete_mailaccount`)
- [x] `mail forwards list` / `mail forwards get <address>` (`get_mailforwards`, with `mail_forward` filter)
- [ ] Mail forward write paths (`add_mailforward`, `update_mailforward`, `delete_mailforward`)
- [x] `mail filters list` (`get_mailstandardfilter`)
- [ ] Mail standard filter write paths (`update_mailstandardfilter`)
- [x] `mail lists list` (`get_mailinglists`)
- [ ] Mailing list write paths (`add_mailinglist`, `update_mailinglist`, `delete_mailinglist`)

## Hosting resources

- [x] `databases list` / `databases get <database-login>` (`get_databases`, with `database_login` filter)
- [ ] Database write paths (`add_database`, `update_database`, `delete_database`)
- [x] `ftpusers list` / `ftpusers get <ftp-login>` (`get_ftpusers`, with `ftp_login` filter)
- [ ] FTP user write paths (`add_ftpuser`, `update_ftpuser`, `delete_ftpuser`)
- [ ] Samba users (`get_sambausers`, `add_sambauser`, `update_sambauser`, `delete_sambauser`)
- [ ] DDNS users (`get_ddnsusers`, `add_ddnsuser`, `update_ddnsuser`, `delete_ddnsuser`)
- [ ] Cronjobs (`get_cronjobs`, `add_cronjob`, `update_cronjob`, `delete_cronjob`)
- [ ] Directory protection (`get_directoryprotection`, `add_directoryprotection`, `update_directoryprotection`, `delete_directoryprotection`)
- [ ] Software install entries (`get_softwareinstall`, `add_softwareinstall`, `delete_softwareinstall`)
- [ ] SSL certificate management (`add_lets_encrypt_csr`, `update_ssl_certificate`, …)
- [ ] Filesystem helpers (`chown`, `symlink`)
