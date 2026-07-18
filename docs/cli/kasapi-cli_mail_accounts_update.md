## kasapi-cli mail accounts update

Replace mutable fields of a mail account (update_mailaccount)

```
kasapi-cli mail accounts update <mail-login> [field flags] [flags]
```

### Options

```
      --active string                   replacement mailbox status (Y|N)
      --allow-nets string               replacement IP/CIDR network access restriction (an explicitly empty value clears it)
      --copy-address string             replacement BCC copy recipient address(es)
  -h, --help                            help for update
      --password string                 replacement mailbox password (sent as mail_new_password)
      --responder string                replacement auto-responder setting: "N", "Y", or a "<start>|<end>" timestamp range
      --responder-content-type string   replacement auto-responder body format (html|text)
      --responder-displayname string    replacement auto-responder sender display name
      --responder-text string           replacement auto-responder message body
      --sender-alias string             replacement permitted FROM alias address(es)
      --webmail-autologin string        replacement KAS-to-webmail auto-login setting (Y|N)
      --xlist-archiv string             replacement XLIST archive folder name
      --xlist-drafts string             replacement XLIST drafts folder name
      --xlist-enabled string            replacement XLIST special-folder mapping setting (Y|N)
      --xlist-sent string               replacement XLIST sent-items folder name
      --xlist-spam string               replacement XLIST spam folder name
      --xlist-trash string              replacement XLIST trash folder name
```

### Options inherited from parent commands

```
      --audit-log string                 append a JSON-Lines audit record for each write action to this file (also KAS_AUDIT_LOG); a logfmt line always goes to stderr regardless
      --auth-data string                 KAS auth data (overrides config and KAS_AUTHDATA)
      --auth-type string                 KAS auth strategy: 'plain' = send password on each KasApi call (no KasAuth, no 2FA support); 'session' = bootstrap via KasAuth and reuse the credential token. Overrides config and KAS_AUTHTYPE.
      --config string                    path to the kasapi-cli config file (overrides the default location)
      --dry-run                          preview a write command's KAS request (action + redacted parameters) and exit 0 without dispatching or prompting; honours --output
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

