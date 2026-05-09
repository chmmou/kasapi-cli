# usage

Inspect webspace and traffic counters for the authenticated account.

KAS API: [`get_space`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-space-inc.html), [`get_space_usage`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-space-usage-inc.html), [`get_traffic`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-traffic-inc.html).

## Webspace summary

`usage space` calls `get_space` and returns the rolled-up webspace usage (one row per category — mail, databases, www, backup, ...).

```sh
kasapi-cli usage space
```

```text
CATEGORY  USED_MB  ALLOWED_MB  PERCENT
mail      237.5    1024        23.2
www       4801.3   35600       13.5
db        412.0    81920       0.5
backup    62.0     0           0.0
```

## Webspace per directory

`usage space-detail` calls `get_space_usage` and returns one row per top-level subdirectory under the webspace root. Useful when the summary shows an unexpected delta and you want to find the responsible directory:

```sh
kasapi-cli usage space-detail -o json | jq 'sort_by(-.detail_size_mb)[:10]'
```

## Traffic

`usage traffic` calls `get_traffic` and returns the per-day traffic counters for the current billing window. The first row (`Day == 0`) is the rolled-up summary for the whole window — use the `(Traffic).IsSummary()` helper from `internal/usage` if you consume the data programmatically, or filter on `day != 0` in shell:

```sh
kasapi-cli usage traffic -o json | jq '.[] | select(.day != 0)'
```

## See also

- [`../cli/kasapi-cli_usage.md`](../cli/kasapi-cli_usage.md) and the per-subcommand pages alongside it — flag reference.
