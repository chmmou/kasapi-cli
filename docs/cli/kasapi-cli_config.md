## kasapi-cli config

Inspect and bootstrap the kasapi-cli configuration

### Options

```
  -h, --help   help for config
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
* [kasapi-cli config add-profile](kasapi-cli_config_add-profile.md)	 - Interactively add a new profile to the config file
* [kasapi-cli config init](kasapi-cli_config_init.md)	 - Interactively create or replace a profile in the config file
* [kasapi-cli config list-profiles](kasapi-cli_config_list-profiles.md)	 - List configured profiles and their auth_type (auth_data redacted)
* [kasapi-cli config path](kasapi-cli_config_path.md)	 - Print the resolved config-file path
* [kasapi-cli config show](kasapi-cli_config_show.md)	 - Print the resolved effective config (auth_data redacted)
* [kasapi-cli config use-profile](kasapi-cli_config_use-profile.md)	 - Switch the persistent default_profile and invalidate the outgoing session

