## kasapi-cli

Command-line client for the All-Inkl KAS API

### Synopsis

kasapi-cli is a command-line client for the All-Inkl Kunden-Administrations-System SOAP API.

```
kasapi-cli [flags]
```

### Options

```
      --auth-data string                 KAS auth data (overrides config and KAS_AUTHDATA)
      --auth-type string                 KAS auth strategy: 'plain' = send password on each KasApi call (no KasAuth, no 2FA support); 'session' = bootstrap via KasAuth and reuse the credential token. Overrides config and KAS_AUTHTYPE.
      --config string                    path to the kasapi-cli config file (overrides the default location)
  -h, --help                             help for kasapi-cli
      --login string                     KAS login (overrides config and KAS_LOGIN)
      --otp string                       2FA one-time PIN — sent to KasAuth as session_2fa during the credential-token bootstrap. Requires auth_type=session; the KAS API does not document 2FA on direct kas_auth_type=plain calls.
  -o, --output string                    output format: json|yaml|table (default table)
      --profile string                   profile to select from the config file (overrides default_profile)
      --session-lifetime int             session_lifetime in seconds passed to KasAuth (1..30000); 0 keeps the server default. Requires auth_type=session.
      --session-update-lifetime string   session_update_lifetime passed to KasAuth ('Y' = sliding window, 'N' = fixed). Empty omits the parameter. Requires auth_type=session.
  -v, --verbose                          enable verbose logging on stderr
  -y, --yes                              skip confirmation prompts on destructive operations
```

### SEE ALSO

* [kasapi-cli accounts](kasapi-cli_accounts.md)	 - Inspect KAS accounts owned by the authenticated login
* [kasapi-cli completion](kasapi-cli_completion.md)	 - Generate the autocompletion script for the specified shell
* [kasapi-cli config](kasapi-cli_config.md)	 - Inspect and bootstrap the kasapi-cli configuration
* [kasapi-cli cronjobs](kasapi-cli_cronjobs.md)	 - Inspect cronjobs visible to the login (get_cronjobs)
* [kasapi-cli databases](kasapi-cli_databases.md)	 - Inspect databases visible to the login (get_databases)
* [kasapi-cli ddnsusers](kasapi-cli_ddnsusers.md)	 - Inspect DDNS users visible to the login (get_ddnsusers)
* [kasapi-cli directoryprotection](kasapi-cli_directoryprotection.md)	 - Inspect directory (htaccess) protections (get_directoryprotection)
* [kasapi-cli dns](kasapi-cli_dns.md)	 - Inspect DNS records for a zone
* [kasapi-cli domains](kasapi-cli_domains.md)	 - Inspect domains owned by the authenticated account
* [kasapi-cli ftpusers](kasapi-cli_ftpusers.md)	 - Inspect FTP users visible to the login (get_ftpusers)
* [kasapi-cli mail](kasapi-cli_mail.md)	 - Inspect mail accounts, forwards, filters, and mailing lists
* [kasapi-cli sambausers](kasapi-cli_sambausers.md)	 - Inspect Samba/CIFS users visible to the login (get_sambausers)
* [kasapi-cli server](kasapi-cli_server.md)	 - Inspect the host server kasapi-cli is talking to
* [kasapi-cli sessions](kasapi-cli_sessions.md)	 - Manage KAS session tokens (delete_session)
* [kasapi-cli softwareinstalls](kasapi-cli_softwareinstalls.md)	 - Inspect installable software templates (get_softwareinstall)
* [kasapi-cli subdomains](kasapi-cli_subdomains.md)	 - Inspect subdomains owned by the authenticated account
* [kasapi-cli tlds](kasapi-cli_tlds.md)	 - Inspect the catalog of registrable top-level domains
* [kasapi-cli usage](kasapi-cli_usage.md)	 - Inspect webspace and traffic counters

