# server

Inspect the host server `kasapi-cli` is talking to.

KAS API: [`get_server_information`](https://kasapi.kasserver.com/dokumentation/phpdoc/packages/API%20Functions.html).

## Server info

`server info` returns one row per installed service (mysql, php, apache, mailserver, ...) with the version and any hosting limits the server reports for that service.

```sh
kasapi-cli server info
```

```text
SERVICE     VERSION   PHP_LIMIT  MAIL_LIMIT  ...
mysql       10.5.27
php         8.4.4
apache      2.4.62
mailserver  ...
```

The full per-service field set (some services carry extra keys like `php_max_filesize`, `mail_max_attachment_size`, ...) is preserved in `-o json` / `-o yaml`.

## See also

- [`../cli/kasapi-cli_server.md`](../cli/kasapi-cli_server.md) — flag reference.
