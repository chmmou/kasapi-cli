# dns

List DNS records for a single zone.

KAS API: [`get_dns_settings`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-dns-settings-inc.html).

## List records

`dns list --domain <zone>` is required — `get_dns_settings` always operates on one zone.

```sh
kasapi-cli dns list --domain example.com
```

```text
ID    NAME              TYPE  DATA                AUX
1234  example.com.       A     192.0.2.1           0
1235  www.example.com.   A     192.0.2.1           0
1236  example.com.       MX    mail.example.com.   10
1237  _dmarc.example.com TXT   v=DMARC1; p=reject  0
```

The `AUX` column carries the priority for `MX` / `SRV` records (always 0 for record types that ignore it). For pipelining, JSON output keeps the raw KAS field names (`record_id`, `record_name`, `record_type`, `record_data`, `record_aux`):

```sh
kasapi-cli dns list --domain example.com -o json | jq '.[] | select(.record_type == "MX")'
```

## See also

- [`../cli/kasapi-cli_dns.md`](../cli/kasapi-cli_dns.md), [`../cli/kasapi-cli_dns_list.md`](../cli/kasapi-cli_dns_list.md) — flag reference.
