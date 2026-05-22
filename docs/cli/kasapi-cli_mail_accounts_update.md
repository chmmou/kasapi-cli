## kasapi-cli mail accounts update

Replace mutable fields of a mail account (update_mailaccount)

```
kasapi-cli mail accounts update <mail-login> [field flags] [flags]
```

### Options

```
      --active string                   mailbox status (Y|N)
      --allow-nets string               restrict access to these IP/CIDR networks
      --copy-address string             BCC copy recipient address(es)
  -h, --help                            help for update
      --password string                 replacement mailbox password (sent as mail_new_password)
      --responder string                auto-responder: "N", "Y", or a "<start>|<end>" timestamp range
      --responder-content-type string   auto-responder body format (html|text)
      --responder-displayname string    auto-responder sender display name
      --responder-text string           auto-responder message body
      --sender-alias string             permitted FROM alias address(es)
      --webmail-autologin string        allow KAS-to-webmail auto-login (Y|N)
      --xlist-archiv string             XLIST archive folder name
      --xlist-drafts string             XLIST drafts folder name
      --xlist-enabled string            enable XLIST special-folder mapping (Y|N)
      --xlist-sent string               XLIST sent-items folder name
      --xlist-spam string               XLIST spam folder name
      --xlist-trash string              XLIST trash folder name
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

* [kasapi-cli mail accounts](kasapi-cli_mail_accounts.md)	 - Inspect and manage mail accounts (get/add/update/delete_mailaccount)

