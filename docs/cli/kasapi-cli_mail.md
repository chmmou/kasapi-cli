## kasapi-cli mail

Inspect mail accounts, forwards, filters, and mailing lists

### Options

```
  -h, --help   help for mail
```

### Options inherited from parent commands

```
      --auth-data string                 KAS auth data (overrides config and KAS_AUTHDATA)
      --auth-type string                 KAS auth strategy: 'plain' = send password on each KasApi call (no KasAuth, no 2FA support); 'session' = bootstrap via KasAuth and reuse the credential token. Overrides config and KAS_AUTHTYPE.
      --config string                    path to the kasapi-cli config file (overrides the default location)
      --login string                     KAS login (overrides config and KAS_LOGIN)
      --otp string                       2FA one-time PIN — sent to KasAuth as session_2fa during the credential-token bootstrap. Requires auth_type=session; the KAS API does not document 2FA on direct kas_auth_type=plain calls.
  -o, --output string                    output format: json|yaml|table (default table)
      --profile string                   profile to select from the config file (overrides default_profile)
      --session-lifetime int             session_lifetime in seconds passed to KasAuth (1..30000); 0 keeps the server default. Requires auth_type=session.
      --session-update-lifetime string   session_update_lifetime passed to KasAuth ('Y' = sliding window, 'N' = fixed). Empty omits the parameter. Requires auth_type=session.
  -v, --verbose                          enable verbose logging on stderr
```

### SEE ALSO

* [kasapi-cli](kasapi-cli.md)	 - Command-line client for the All-Inkl KAS API
* [kasapi-cli mail accounts](kasapi-cli_mail_accounts.md)	 - Inspect mail accounts (get_mailaccounts)
* [kasapi-cli mail filters](kasapi-cli_mail_filters.md)	 - Inspect mail standard filters (get_mailstandardfilter)
* [kasapi-cli mail forwards](kasapi-cli_mail_forwards.md)	 - Inspect mail forwards (get_mailforwards)
* [kasapi-cli mail lists](kasapi-cli_mail_lists.md)	 - Inspect mailing lists (get_mailinglists)

