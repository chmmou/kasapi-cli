# accounts

Inspect KAS accounts visible to the authenticated login.

KAS API: [`get_accounts`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-accounts-inc.html), [`get_accountsettings`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-accountsettings-inc.html), [`get_accountresources`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-accountresources-inc.html).

## List accounts

`accounts list` calls `get_accounts` without a filter — for a main login this is every sub-account, for a sub-account it is just the login itself.

```sh
kasapi-cli accounts list
```

```text
LOGIN     COMMENT             MAIL                            MAX_DOMAIN  MAX_WEBSPACE  USED_MB  2FA        IN_PROGRESS
w0000001  main account        admin@example.com               4           35600         4801.3   Y          FALSE
w0000002  Git mirror          admin@example.com               0           8192          329.1    inherited  FALSE
```

JSON / YAML output preserves the full set of fields — including `account_password` (always empty in the live API) and timestamps — for piping into `jq`:

```sh
kasapi-cli accounts list -o json | jq '.[] | {login: .account_login, used: .account_usedspace}'
```

## Get a single account

`accounts get <login>` reuses `get_accounts` with an `account_login` filter and unwraps the single-entry result.

```sh
kasapi-cli accounts get w0000001 -o yaml
```

## Settings

`accounts settings` calls `get_accountsettings` and shows the per-account preferences (timezone, language, KAS panel options).

## Resources / quota

`accounts resources` calls `get_accountresources` and shows quota counters (used vs. allowed for domains, mail accounts, databases, FTP users, ...).

## See also

- [`../cli/kasapi-cli_accounts.md`](../cli/kasapi-cli_accounts.md) — flag reference.
