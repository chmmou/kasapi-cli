# Destructive-write safety contract

The write surface of the KAS API is irreversible: `delete_*`, `reset_*`,
`move_*`, `update_chown` and similar actions cannot be undone from the
client. `kasapi-cli` therefore gates every destructive subcommand behind
an explicit confirmation before the SOAP call is dispatched.

This page documents the behavioural contract (issue
[#109](https://github.com/chmmou/kasapi-cli/issues/109), part of the
v0.2.0 write phase, [#13](https://github.com/chmmou/kasapi-cli/issues/13)).
The read commands shipping today are non-destructive and are not gated.

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

## Previewing instead of confirming

A `--dry-run` flag that prints the exact KAS action and parameter map
without dispatching anything is planned separately
([#132](https://github.com/chmmou/kasapi-cli/issues/132)); it will
short-circuit this prompt because nothing destructive happens.
