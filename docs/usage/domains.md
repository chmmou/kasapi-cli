# domains

Inspect domains, subdomains, and the catalog of registrable top-level domains.

KAS API: [`get_domains`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-domains-inc.html), [`get_subdomains`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-subdomains-inc.html), [`get_topleveldomains`](https://kasapi.kasserver.com/dokumentation/phpdoc/files/get-topleveldomains-inc.html).

## List domains

```sh
kasapi-cli domains list
```

```text
DOMAIN              PATH                 TYPE  EXPIRES     SSL  IS_PROTECTED
example.com         /www/example.com     own   2027-08-14  Y    N
mirror.example.com  /www/example.com     own   2027-08-14  Y    N
```

The list view collapses the per-zone metadata (registrar, domain pointer target) into one row per domain. JSON / YAML keeps every field that `get_domains` returned — including `domain_dnsmaster` and `domain_renew_status`.

## Get a single domain

`domains get <domain>` calls `get_domains` with `domain_name` and unwraps the single-entry result. Useful when you want the full record without filtering client-side.

```sh
kasapi-cli domains get example.com -o yaml
```

## List subdomains

```sh
kasapi-cli subdomains list
kasapi-cli subdomains get www.example.com -o json
```

## Registrable TLDs

`tlds list` calls `get_topleveldomains` and returns the catalog of TLDs the All-Inkl registrar can register, with their per-year price and whether they support transfer / DNSSEC.

```sh
kasapi-cli tlds list -o json | jq '.[] | select(.tld_dnssec_supported == "Y") | .tld_name'
```

## See also

- [`../cli/kasapi-cli_domains.md`](../cli/kasapi-cli_domains.md), [`../cli/kasapi-cli_subdomains.md`](../cli/kasapi-cli_subdomains.md), [`../cli/kasapi-cli_tlds.md`](../cli/kasapi-cli_tlds.md) — flag reference.
