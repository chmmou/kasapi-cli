## kasapi-cli ddnsusers

Inspect DDNS users visible to the login (get_ddnsusers)

### Options

```
  -h, --help   help for ddnsusers
```

### Options inherited from parent commands

```
      --auth-data string                 KAS auth data (overrides config and KAS_AUTHDATA)
      --auth-type string                 KAS auth strategy: 'plain' = send password on each KasApi call (no KasAuth, no 2FA support); 'session' = bootstrap via KasAuth and reuse the credential token. Overrides config and KAS_AUTHTYPE.
      --config string                    path to the kasapi-cli config file (overrides the default location)
      --login string                     KAS login (overrides config and KAS_LOGIN)
      --no-color                         disable coloured output
      --otp string                       2FA one-time PIN — sent to KasAuth as session_2fa during the credential-token bootstrap. Requires auth_type=session; the KAS API does not document 2FA on direct kas_auth_type=plain calls.
  -o, --output string                    output format: json|yaml|table (default table)
      --profile string                   profile to select from the config file (overrides default_profile)
      --session-lifetime int             session_lifetime in seconds passed to KasAuth (1..30000); 0 keeps the server default. Requires auth_type=session.
      --session-update-lifetime string   session_update_lifetime passed to KasAuth ('Y' = sliding window, 'N' = fixed). Empty omits the parameter. Requires auth_type=session.
  -v, --verbose                          enable verbose logging on stderr
  -y, --yes                              skip confirmation prompts on destructive operations
```

### SEE ALSO

* [kasapi-cli](kasapi-cli.md)	 - Command-line client for the All-Inkl KAS API
* [kasapi-cli ddnsusers get](kasapi-cli_ddnsusers_get.md)	 - Show details for a single DDNS user (get_ddnsusers with ddns_login)
* [kasapi-cli ddnsusers list](kasapi-cli_ddnsusers_list.md)	 - List all DDNS users (get_ddnsusers, no filter)

