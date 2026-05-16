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

## Previewing instead of confirming

A `--dry-run` flag that prints the exact KAS action and parameter map
without dispatching anything is planned separately
([#132](https://github.com/chmmou/kasapi-cli/issues/132)); it will
short-circuit this prompt because nothing destructive happens.
