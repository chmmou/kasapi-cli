# mail

Inspect mail accounts, forwards, filters, and mailing lists.

KAS API: [`get_mailaccounts`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-mailaccounts-inc.html), [`get_mailforwards`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-mailforwards-inc.html), [`get_mailstandardfilter`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-mailstandardfilter-inc.html), [`get_mailinglists`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-mailinglists-inc.html).

## Mail accounts

```sh
kasapi-cli mail accounts list
kasapi-cli mail accounts get user@example.com -o yaml
```

```text
ADDRESS              QUOTA_MB  USED_MB  IN_PROGRESS
user@example.com     1024      237.5    FALSE
postmaster@example.com  1024   12.1     FALSE
```

## Mail forwards

```sh
kasapi-cli mail forwards list
kasapi-cli mail forwards get info@example.com
```

```text
SOURCE              TARGETS
info@example.com    admin@example.com, ops@example.com
sales@example.com   sales-team@partner.com
```

The `TARGETS` column flattens `mail_forward_target_*` into one comma-joined list; JSON output keeps the original numbered targets so downstream tools can match the KAS API contract exactly.

## Mail filters

`mail filters list` calls `get_mailstandardfilter` and returns the **catalog of pre-defined filter presets** an account can attach via the `mail_spamfilter` setting on a mailaccount or forward — `rspamd`, the SpamAssassin variants, the virus scanner modes, and the configured spam-database lookups. Each row carries the preset id, its category (`rspamd`, `content`, `spamc`, `virus`, `reject`), a human-readable title, and the All-Inkl `recommended` flag. The KAS endpoint takes no parameters and is read-only — adding a preset to a mailbox happens via the (not-yet-implemented) `add_mailstandardfilter` write endpoint, tracked in the v0.2.0 backlog.

## Mailing lists

```sh
kasapi-cli mail lists list
kasapi-cli mail lists get newsletter@example.com -o json
```

The list view shows one row per list with the subscriber count; `get` returns the full subscriber set under `mailinglist_members`.

## See also

- [`../cli/kasapi-cli_mail.md`](../cli/kasapi-cli_mail.md) and the per-subcommand pages alongside it — flag reference.
