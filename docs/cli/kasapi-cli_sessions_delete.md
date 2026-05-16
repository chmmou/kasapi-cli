## kasapi-cli sessions delete

Invalidate the active profile's cached session token (delete_session)

### Synopsis

Invalidate the resolved profile's cached session token, both server-side (kas_action=delete_session) and in the local sessions.toml cache.

Acts on the *currently cached* token only; it never bootstraps a fresh token just to delete it. Idempotent: a missing or already-invalid session is reported and exits 0. No confirmation prompt — deleting a session merely forces a re-authentication on the next session-mode call; the global --yes flag has no effect here because there is nothing to confirm.

```
kasapi-cli sessions delete [flags]
```

### Options

```
  -h, --help   help for delete
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

* [kasapi-cli sessions](kasapi-cli_sessions.md)	 - Manage KAS session tokens (delete_session)

