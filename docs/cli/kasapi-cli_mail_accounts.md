## kasapi-cli mail accounts

Inspect mail accounts (get_mailaccounts)

### Options

```
  -h, --help   help for accounts
```

### Options inherited from parent commands

```
      --audit-log string                 append a JSON-Lines audit record for each write action to this file (also KAS_AUDIT_LOG); a logfmt line always goes to stderr regardless
      --auth-data string                 KAS auth data (overrides config and KAS_AUTHDATA)
      --auth-type string                 KAS auth strategy: 'plain' = send password on each KasApi call (no KasAuth, no 2FA support); 'session' = bootstrap via KasAuth and reuse the credential token. Overrides config and KAS_AUTHTYPE.
      --config string                    path to the kasapi-cli config file (overrides the default location)
      --dry-run                          preview a destructive command's KAS request (action + redacted parameters) and exit 0 without dispatching or prompting; honours --output
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

* [kasapi-cli mail](kasapi-cli_mail.md)	 - Inspect mail accounts and filters; inspect and manage forwards and mailing lists
* [kasapi-cli mail accounts get](kasapi-cli_mail_accounts_get.md)	 - Show details for a single mail account (get_mailaccounts with mail_login)
* [kasapi-cli mail accounts list](kasapi-cli_mail_accounts_list.md)	 - List all mail accounts (get_mailaccounts)

