# Destructive-write safety contract

The write surface of the KAS API is irreversible: `delete_*`, `reset_*`,
`move_*`, `update_chown` and similar actions cannot be undone from the
client. `kasapi-cli` therefore gates every destructive subcommand behind
an explicit confirmation before the SOAP call is dispatched.

This page documents the behavioural contract (issue
[#109](https://github.com/chmmou/kasapi-cli/issues/109), part of the
v0.2.0 write phase, [#13](https://github.com/chmmou/kasapi-cli/issues/13)).
The read commands are non-destructive and are not gated.

The first write endpoints —
[`mail forwards`](https://github.com/chmmou/kasapi-cli/issues/115)
`add` / `update` / `delete` — wire this contract. `delete` and
`update` are gated by the confirmation prompt (an `update_mailforward`
replaces the whole target list and is therefore irreversible); `add`
creates and is reversible, so it is **not** prompted. All three honour
`--dry-run` and emit an audit record.

[`mail lists`](https://github.com/chmmou/kasapi-cli/issues/117) `add` /
`update` / `delete` wire the same contract with the same policy:
`update` (it replaces the subscriber / restrict-post / config fields
wholesale) and `delete` are gated; `add` is reversible and not
prompted. The list password passed to `add` is redacted in the
`--dry-run` preview and the audit record.

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
