## kasapi-cli mail filters add

Set the standard-filter chain on a mail account (add_mailstandardfilter)

### Synopsis

Set the configured standard-filter chain on a mail account via
add_mailstandardfilter. Each --filter is one item of the chain, either a
bare filter id (e.g. "pdw") or "<filter-id>:<option>=<value>" (e.g.
"spamc_move:move=Spam"). Items are joined with ';' on the wire and the
chain replaces what was configured before — there is no per-item add.
Use "mail filters list" for the available filter ids.

```
kasapi-cli mail filters add <mail-login> --filter <item> [--filter <item>...] [flags]
```

### Options

```
      --filter stringArray   filter chain item (repeatable; replaces the full chain)
  -h, --help                 help for add
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

* [kasapi-cli mail filters](kasapi-cli_mail_filters.md)	 - Inspect and manage mail standard filters (get/add/delete_mailstandardfilter)

