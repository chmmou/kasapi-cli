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

- [x] `account get` (`get_accounts`, `get_accountsettings`, `get_accountresources`)
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

- [ ] Mail accounts (`get_mailaccounts`, `add_mailaccount`, `update_mailaccount`, `delete_mailaccount`)
- [ ] Mail forwards (`get_mailforwards`, `add_mailforward`, `update_mailforward`, `delete_mailforward`)
- [ ] Mail standard filters (`get_mailstandardfilter`, `update_mailstandardfilter`)
- [ ] Mailing lists (`get_mailinglists`, `add_mailinglist`, `update_mailinglist`, `delete_mailinglist`)

## Hosting resources

- [ ] Databases (`get_databases`, `add_database`, `update_database`, `delete_database`)
- [ ] FTP users (`get_ftpusers`, `add_ftpuser`, `update_ftpuser`, `delete_ftpuser`)
- [ ] Samba users (`get_sambausers`, `add_sambauser`, `update_sambauser`, `delete_sambauser`)
- [ ] DDNS users (`get_ddnsusers`, `add_ddnsuser`, `update_ddnsuser`, `delete_ddnsuser`)
- [ ] Cronjobs (`get_cronjobs`, `add_cronjob`, `update_cronjob`, `delete_cronjob`)
- [ ] Directory protection (`get_directoryprotection`, `add_directoryprotection`, `update_directoryprotection`, `delete_directoryprotection`)
- [ ] Software install entries (`get_softwareinstall`, `add_softwareinstall`, `delete_softwareinstall`)
- [ ] SSL certificate management (`add_lets_encrypt_csr`, `update_ssl_certificate`, …)
- [ ] Filesystem helpers (`chown`, `symlink`)
