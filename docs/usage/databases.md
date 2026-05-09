# databases

Inspect MariaDB / MySQL databases visible to the login.

KAS API: [`get_databases`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-databases-inc.html).

## List databases

```sh
kasapi-cli databases list
```

```text
NAME              COMMENT       SIZE_MB  USER          IN_PROGRESS
d0123456_main     production    412.0    d0123456      FALSE
d0123456_staging  staging dump  18.7     d0123456      FALSE
```

## Get a single database

`databases get <name>` reuses `get_databases` with a `database_name` filter and unwraps the single-entry result. Use it when you want to fetch the full record (charset, collation, last-modified) without re-filtering client-side:

```sh
kasapi-cli databases get d0123456_main -o yaml
```

JSON output keeps the original `database_*` keys so output can be merged with backup tooling that reads the KAS contract directly.

## See also

- [`../cli/kasapi-cli_databases.md`](../cli/kasapi-cli_databases.md) — flag reference.
