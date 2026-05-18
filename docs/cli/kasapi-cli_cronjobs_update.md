## kasapi-cli cronjobs update

Replace mutable fields of a cronjob (update_cronjob)

```
kasapi-cli cronjobs update <cronjob-id> [schedule/mail flags] [flags]
```

### Options

```
      --active                  whether the cronjob is active; pass --active=false to disable (default true)
      --comment string          cronjob comment / label (required for add)
      --day-of-month string     schedule day-of-month field (default "*")
      --day-of-week string      schedule day-of-week field (0-7, Sun=0|7) (default "*")
  -h, --help                    help for update
      --hour string             schedule hour field (default "*")
      --http-password string    HTTP basic-auth password for the call
      --http-user string        HTTP basic-auth user for the call
      --mail-address string     notification mail address (mail_adress)
      --mail-condition string   when to send the notification mail
      --mail-subject string     notification mail subject (default|comment) (default "default")
      --minute string           schedule minute field (default "*")
      --month string            schedule month field (default "*")
      --protocol string         request protocol (http|https) (default "https")
      --url string              URL to call (http_url; required for add)
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

* [kasapi-cli cronjobs](kasapi-cli_cronjobs.md)	 - Inspect and manage cronjobs (get/add/update/delete_cronjob)

