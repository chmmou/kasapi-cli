package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Confirm prompts the user on out with prompt + " [y/N]: " and reads a
// reply from in. It returns true only on an explicit "y" or "yes"
// (case-insensitive); empty input or any other string returns false.
//
// Confirm is intended for destructive write operations (#13). When the
// global --yes flag is set, callers should skip the prompt entirely
// rather than feeding the answer through this helper.
func Confirm(in io.Reader, out io.Writer, prompt string) (bool, error) {
	if _, err := fmt.Fprintf(out, "%s [y/N]: ", prompt); err != nil {
		return false, err
	}
	r := bufio.NewReader(in)
	line, err := r.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// stdinIsTTY reports whether the process stdin is an interactive
// terminal. It is the single source of truth for TTY detection in this
// package (config init/add-profile and the destructive-write gate all
// share it) so the platform-specific cast lives in exactly one place.
func stdinIsTTY() bool {
	//nolint:gosec // G115: file descriptors fit in int on every platform Go targets; term.IsTerminal takes int.
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// ErrConfirmationDeclined is returned by GateDestructive when the user
// answered the [y/N] prompt with anything other than yes. It is wrapped
// in an ExitError(ExitUserError); callers may errors.Is against it to
// distinguish a deliberate abort from an input/transport failure.
var ErrConfirmationDeclined = errors.New("cli: destructive operation cancelled by user")

// ErrConfirmationRequired is returned by GateDestructive when stdin is
// not an interactive terminal and --yes was not given, so no prompt can
// be shown. It maps to ExitUserError: the caller must either run
// interactively or pass --yes to proceed non-interactively.
var ErrConfirmationRequired = errors.New("cli: refusing destructive operation: stdin is not a TTY and --yes was not given")

// ConfirmAction is the one-line description of a pending destructive
// change, shown to the user before the [y/N] prompt: "About to <Verb>
// <Resource> \"<ID>\". This cannot be undone."
type ConfirmAction struct {
	Verb     string // imperative, e.g. "delete", "reset", "move"
	Resource string // human resource type, e.g. "mail account", "DNS record"
	ID       string // identifier of the target resource
}

// Summary renders the one-line description shown to the user before
// the [y/N] prompt. Exported so tests can pin the rendered prompt
// (and the loudness verb each slice chose) without instantiating a
// real terminal.
func (a ConfirmAction) Summary() string {
	return fmt.Sprintf("About to %s %s %q. This cannot be undone.", a.Verb, a.Resource, a.ID)
}

// GateDestructive enforces the confirmation contract for a destructive
// command before its SOAP call is dispatched. yes is opts.Yes; isTTY is
// stdinIsTTY() (injected so tests need no real terminal).
//
//   - yes == true            → proceed silently (automation opted in).
//   - !isTTY (and !yes)      → ErrConfirmationRequired, no prompt.
//   - interactive, declined  → ErrConfirmationDeclined.
//   - interactive, accepted  → nil.
//
// The non-nil returns are *ExitError(ExitUserError) so CodeFor maps a
// declined or impossible confirmation to exit code 1.
func GateDestructive(in io.Reader, out io.Writer, isTTY, yes bool, a ConfirmAction) error {
	if yes {
		return nil
	}
	if !isTTY {
		return UserError(ErrConfirmationRequired, "")
	}
	ok, err := Confirm(in, out, a.Summary())
	if err != nil {
		return UserError(err, "confirm")
	}
	if !ok {
		return UserError(ErrConfirmationDeclined, "")
	}
	return nil
}
