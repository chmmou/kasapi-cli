# Destructive-write safety contract

The write surface of the KAS API is irreversible: `delete_*`, `reset_*`,
`move_*`, `update_chown` and similar actions cannot be undone from the
client. `kasapi-cli` therefore gates every destructive subcommand behind
an explicit confirmation before the SOAP call is dispatched.

This page documents the behavioural contract (issue
[#109](https://github.com/chmmou/kasapi-cli/issues/109), part of the
v0.2.0 write phase, [#13](https://github.com/chmmou/kasapi-cli/issues/13)).
The read commands are non-destructive and are not gated.

## Per-slice baseline

Every wired write slice carries the same baseline policy. Listing it
once here keeps the per-slice notes below short — they only call out
the deviations.

- `add` is reversible (creates a new resource) → **not** gated; no
  prompt.
- `update` and `delete` are irreversible (wholesale field replacement
  on `update`; resource removal on `delete`) → **gated** by the #109
  prompt. The prompt phrases `update` as replacing the resource's
  *settings*.
- `update` sends **only the explicitly-changed flags** (keyed on
  cobra `Changed`), so an empty value is a deliberate "clear", not
  "leave unchanged".
- All three subcommands honour `--dry-run` (#132) and emit a #131
  audit record. Secret request parameters are redacted in **both**
  sinks.
- Subcommands whose KAS action generates the login server-side
  (`add_ftpuser`, `add_sambauser`, `add_ddnsuser`, `add_database`,
  `add_mailinglist`, `add_mailforward`, `add_mailaccount`) print the
  generated identifier on success.

## Per-slice deviations

| Slice (issue) | Deviation from the baseline above |
|---|---|
| [`mail forwards`](https://github.com/chmmou/kasapi-cli/issues/115) | First slice to wire #109/#131/#132. `update_mailforward` replaces the whole *target list*; the prompt phrases this as replacing the forward's *targets* (not "settings"). |
| [`mail lists`](https://github.com/chmmou/kasapi-cli/issues/117) | The list password passed to `add` is redacted in the dry-run preview and audit record. |
| [`cronjobs`](https://github.com/chmmou/kasapi-cli/issues/118) | Any `--http-password` passed to `add`/`update` is redacted. |
| [`ftpusers`](https://github.com/chmmou/kasapi-cli/issues/119) | Password key splits between actions: `--password` → `ftp_password` on add, `ftp_new_password` on update. |
| [`sambausers`](https://github.com/chmmou/kasapi-cli/issues/120) | Password key splits between actions: `--password` → `samba_password` on add, `samba_new_password` on update. Note: the KAS docs wrongly list `samba_new_password` for the create call; the captured fixture confirms the real key is `samba_password`. |
| [`databases`](https://github.com/chmmou/kasapi-cli/issues/122) | **Louder delete prompt**: `delete_database` uses the verb `"permanently delete"` (vs the bare `"delete"` every other slice uses) because the action drops the database AND every row in it — the loudest data-loss surface of the v0.2.0 write phase. Password key splits between actions: `--password` → `database_password` on add, `database_new_password` on update. `--allowed-hosts` is **optional**: an empty value is the KAS API's documented "any host may connect" wildcard, sent verbatim on the wire. |
| [`ddnsusers`](https://github.com/chmmou/kasapi-cli/issues/121) | **No `_new_password` split**: `--password` maps to `dyndns_password` on both `add` and `update`. `update_ddnsuser` accepts `--target-ipv4` / `--target-ipv6` instead of `add`'s legacy `--target-ip`; the ipv4/ipv6 keys are undocumented in the KAS API docs but verified to work against the live system (the captured update request fixture is authoritative). |
| [`mail accounts`](https://github.com/chmmou/kasapi-cli/issues/114) | **Louder delete prompt**: `delete_mailaccount` uses `"permanently delete"` (shared with `databases`) — it drops the mailbox AND every message in it. `add` splits the address on the last `@` into `local_part` / `domain_part`; `update`/`delete` address the account by its generated `mail_login`. Password key splits between actions: `--password` → `mail_password` on add, `mail_new_password` on update. `add`'s Y/N/text toggles and XLIST folder names default to the KAS API's own defaults; `--responder` is passed through verbatim (`N`, `Y`, or a `<start>\|<end>` timestamp range). |
| [`mail filters`](https://github.com/chmmou/kasapi-cli/issues/116) | **Both `add` and `delete` are gated**, not just `delete`: there is no `update_mailstandardfilter`, so `add_mailstandardfilter` *replaces* the configured chain wholesale (any previously-set items not in the new `--filter` list are dropped). Repeatable `--filter <item>` is joined with `;` on the wire (items must not contain `;` and not be empty). `delete_mailstandardfilter` takes only `<mail-login>` and removes the whole chain in one shot; the prompt phrases this as *"remove all standard filters of mail account &lt;login&gt;"* to make the all-at-once effect explicit. **Known API quirk**: `delete` sometimes returns an envelope-level SOAP fault (an internal `sizeof()` PHP error) even when the chain was in fact removed on the server. The fault is surfaced verbatim — if you see it, verify the actual outcome via `kasapi-cli mail accounts get <login>` (the configured chain is reported in the `mail_spamfilter` field). |
| [`directoryprotection`](https://github.com/chmmou/kasapi-cli/issues/123) | **Composite `(path, user)` identity**: `add`/`update`/`delete` take two positional args `<path> <user>` (a single path can protect several users), not one server-generated login — so the `add` identifier is user-supplied and not echoed back. **No `_new_password` split**: `--password` maps to `directory_password` on both `add` and `update`. `--authname` (the htaccess realm label) is optional and always sent on `add`; `update` sends only the changed `--password`/`--authname` (sparse), so an omitted password keeps the current one. KAS also accepts parallel `directory_user`/`directory_password` arrays to create several users in one call (hence the `directory_user_count_neq_passcount` fault); only the captured scalar single-user form is modelled. |

## The contract

Before a destructive call leaves the machine, the command prints a
one-line summary of the pending change and asks for confirmation:

```
About to delete mail account "m0000001". This cannot be undone.
Proceed? [y/N]:
```

- Only an explicit `y` / `yes` (case-insensitive) proceeds. Empty input
  or anything else aborts.
- Declining exits with code **1** (user error); nothing is sent to KAS.

## `--yes` / `-y`

The global `--yes` (`-y`) flag bypasses the prompt for automation that
has already decided to proceed:

```sh
kasapi-cli <destructive-command> --yes
```

## Non-interactive stdin

If stdin is not a terminal (pipe, cron, CI) **and** `--yes` was not
given, the command refuses with exit code **1** rather than blocking on
a prompt that can never be answered. Pass `--yes` to run such a command
non-interactively.

| stdin | `--yes` | result |
|-------|---------|--------|
| TTY   | no      | prompt; proceed only on `y`/`yes`, else exit 1 |
| TTY   | yes     | proceed without prompting |
| not a TTY | no  | exit 1 (`refusing destructive operation`) |
| not a TTY | yes | proceed without prompting |

## Audit log

Independently of the confirmation prompt and of `--verbose`, every
dispatched write action leaves a structured trace
([#131](https://github.com/chmmou/kasapi-cli/issues/131)). The record is
emitted **after** the SOAP call returns, regardless of success or
failure.

A `logfmt`-style line always goes to **stderr**:

```
ts=2026-05-16T12:00:00Z login=w0000001 action=delete_dns_settings target="record 42" outcome=success record_id=42
```

`outcome` is `success`, `failure:<kas_code>` for a typed KAS fault, or a
bare `failure` for a transport/decode error.

Passing `--audit-log <path>` (or setting `KAS_AUDIT_LOG`; the flag wins)
additionally appends the same record as one JSON object per line
(JSON Lines) to that file, which is created with mode `0600`.

Secret request parameters (`auth_data`, `*password`, `*token`,
`*secret`, …) are replaced with `<redacted>` in **both** sinks and never
written. Read commands produce no audit record.

## `--dry-run`: preview without dispatching

`--dry-run` runs every step of a destructive command **except** the
SOAP call ([#132](https://github.com/chmmou/kasapi-cli/issues/132)).
Credentials and the request are resolved exactly as for a real call,
the KAS action and its parameter map are printed, and the command exits
**0** without contacting KAS.

- It **short-circuits the confirmation prompt** — nothing destructive
  happens, so there is nothing to confirm. `--dry-run` together with
  `--yes` still only previews (exit 0, no prompt, no dispatch).
- The preview honours `--output` (`table` default, `json`, `yaml`) so
  it is grep-/jq-/yq-friendly.
- Secret parameters are redacted with the **same** rule as the audit
  log (`<redacted>`).
- An audit record is still emitted, with `outcome=dry-run`, so the
  trace exists and is distinguishable from a real call.

```console
$ kasapi-cli mail forwards delete info@example.de --dry-run
FIELD               VALUE
action              delete_mailforward
target              info@example.de
param.mail_forward  info@example.de

$ kasapi-cli mail forwards add info@example.de --target a@b.de --dry-run -o json
{
  "action": "add_mailforward",
  "target": "info@example.de",
  "params": {
    "domain_part": "example.de",
    "local_part": "info",
    "target_0": "a@b.de"
  }
}
```

`add` is not prompted but `--dry-run` and the audit record still apply
to it, exactly as for the gated `update` / `delete`.
