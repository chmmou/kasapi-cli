# hosting

Read-only inspection of the various hosting features that hang off an All-Inkl webspace: FTP / Samba users, cronjobs, directory protection, software installs, and DDNS users. Grouped because they share the same shape (one resource type per top-level command, almost always with `list` + `get`).

KAS API: [`get_ftpusers`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-ftpusers-inc.html), [`get_sambausers`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-sambausers-inc.html), [`get_cronjobs`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-cronjobs-inc.html), [`get_directoryprotection`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-directoryprotection-inc.html), [`get_softwareinstall`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-softwareinstall-inc.html), [`get_ddnsusers`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-ddnsusers-inc.html).

## FTP users

```sh
kasapi-cli ftpusers list
kasapi-cli ftpusers get f0123456 -o yaml
```

`ftp_user_password` is omitted from the table view but kept in JSON / YAML.

## Samba (CIFS) users

```sh
kasapi-cli sambausers list
kasapi-cli sambausers get s0123456
```

## Cronjobs

```sh
kasapi-cli cronjobs list
kasapi-cli cronjobs get 42 -o yaml
```

The list view collapses the `cronjob_minute` / `cronjob_hour` / `cronjob_day` / `cronjob_month` / `cronjob_dayofweek` quintuple into a single `SCHEDULE` column in cron-format. JSON keeps the individual fields so they can be passed back to a write endpoint unmodified.

## Directory protection (htaccess)

`directoryprotection list` returns one row per `(directory_path, directory_user)` tuple — a directory protected for `N` users surfaces as `N` rows. Optional `--path PATH` filter narrows to a single directory:

```sh
kasapi-cli directoryprotection list
kasapi-cli directoryprotection list --path /www/admin
```

`directory_password` is omitted from the table view but kept in JSON / YAML.

## Software installs

```sh
kasapi-cli softwareinstalls list
kasapi-cli softwareinstalls get 7 -o yaml
```

The KAS endpoint name is `get_softwareinstall` (singular) for both list and get; this client maps it to the plural `softwareinstalls` subcommand to match the rest of the CLI surface. The `image` field carries a base64 data URI, which is stripped from the table view but kept on the struct for JSON / YAML round-trips.

## DDNS users

```sh
kasapi-cli ddnsusers list
kasapi-cli ddnsusers get host.dyn.example.com
```

The list view joins `dyndns_label` and `dyndns_zone` into a single `FQDN` column so the table reflects the hostname clients will look up; `dyndns_password` is omitted from the table view but kept in JSON / YAML.

> [!NOTE]
> The `get` parameter for DDNS is `ddns_login` (no `y`), even though the response keys use the `dyndns_*` prefix — this matches the KAS API contract documented at `get-ddnsusers-inc.html`.

## See also

- [`../cli/kasapi-cli_ftpusers.md`](../cli/kasapi-cli_ftpusers.md), [`../cli/kasapi-cli_sambausers.md`](../cli/kasapi-cli_sambausers.md), [`../cli/kasapi-cli_cronjobs.md`](../cli/kasapi-cli_cronjobs.md), [`../cli/kasapi-cli_directoryprotection.md`](../cli/kasapi-cli_directoryprotection.md), [`../cli/kasapi-cli_softwareinstalls.md`](../cli/kasapi-cli_softwareinstalls.md), [`../cli/kasapi-cli_ddnsusers.md`](../cli/kasapi-cli_ddnsusers.md) — flag reference.
