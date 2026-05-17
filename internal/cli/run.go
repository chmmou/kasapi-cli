package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
)

// runListE returns a cobra RunE closure that follows the canonical
// read-subcommand shape: build the API client, run the supplied
// fetch, wrap a transport error as APIError(action), and render the
// result. It is the no-positional-argument variant — used by `list`,
// `info`, `space`, `space-detail`, `settings`, `resources`, and the
// other no-arg subcommands.
//
// fetch owns the module client construction and the actual call;
// callers typically pass a one-line closure such as
//
//	func(c *api.Client, ctx context.Context) (ddns.DDNSUserList, error) {
//	    return ddns.NewClient(c).List(ctx)
//	}
//
// Flag-derived parameters are passed in via the closure's capture.
func runListE[T any](
	opts *RootOptions,
	action string,
	fetch func(c *api.Client, ctx context.Context) (T, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		c, err := BuildAPIClient(opts)
		if err != nil {
			return err
		}
		v, err := fetch(c, cmd.Context())
		if err != nil {
			return APIError(err, action)
		}
		if err := Render(cmd.OutOrStdout(), opts.Output, v); err != nil {
			return UserError(err, "render")
		}
		return nil
	}
}

// runGetE is the cobra.ExactArgs(1) variant of runListE — fetch
// additionally receives args[0]. The standard caller passes the
// module's Client.Get method curried through a closure:
//
//	func(c *api.Client, ctx context.Context, arg string) (ddns.DDNSUser, error) {
//	    return ddns.NewClient(c).Get(ctx, arg)
//	}
func runGetE[T any](
	opts *RootOptions,
	action string,
	fetch func(c *api.Client, ctx context.Context, arg string) (T, error),
) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		c, err := BuildAPIClient(opts)
		if err != nil {
			return err
		}
		v, err := fetch(c, cmd.Context(), args[0])
		if err != nil {
			return APIError(err, action)
		}
		if err := Render(cmd.OutOrStdout(), opts.Output, v); err != nil {
			return UserError(err, "render")
		}
		return nil
	}
}

// writeSpec describes one dispatched write action. build (see
// runWriteE) turns the parsed args/flags into it: the KAS action name,
// whether it is destructive (→ #109 confirmation gate vs. a plain
// audited write), the ConfirmAction/target for the prompt and audit
// trace, the request parameter map, and the dispatch closure that runs
// the module client method and returns the human success line.
type writeSpec struct {
	action      string
	destructive bool
	confirm     ConfirmAction
	params      map[string]any
	dispatch    func(c *api.Client, ctx context.Context) (string, error)
}

// runWriteE is the write-subcommand counterpart of runListE/runGetE and
// the single seam every write command flows through. It resolves the
// login (for the audit trace), opens the optional --audit-log sink,
// consults ResolveDestructive (gated) or ResolveWrite (non-gated)
// depending on spec.destructive — which also handles --dry-run (#132)
// and the dry-run audit (#131) — and only then builds the API client
// and dispatches. After dispatch it always emits the success/failure
// audit record via OutcomeFor, then prints the result line on success.
func runWriteE(opts *RootOptions, build func(args []string) (writeSpec, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		spec, err := build(args)
		if err != nil {
			return UserError(err, "")
		}

		creds, err := resolveCreds(opts)
		if err != nil {
			return err
		}

		var auditFile io.Writer
		if path := AuditLogPath(opts); path != "" {
			f, oerr := OpenAuditFile(path)
			if oerr != nil {
				return UserError(oerr, "")
			}
			defer func() { _ = f.Close() }()
			auditFile = f
		}

		out := cmd.OutOrStdout()
		stderr := cmd.ErrOrStderr()

		var resolve WriteResolver = ResolveWrite
		if spec.destructive {
			resolve = ResolveDestructive
		}
		proceed, err := resolve(opts, cmd.InOrStdin(), out, stderr, auditFile, stdinIsTTY(),
			creds.Login, spec.action, spec.confirm, spec.params)
		if !proceed {
			// --dry-run → (false, nil): exit 0; declined/refused → (false, err).
			return err
		}

		c, err := BuildAPIClient(opts)
		if err != nil {
			return err
		}
		result, derr := spec.dispatch(c, cmd.Context())

		rec := AuditRecord{
			Time:    time.Now().UTC(),
			Login:   creds.Login,
			Action:  spec.action,
			Target:  spec.confirm.ID,
			Outcome: OutcomeFor(derr),
			Fields:  RedactParams(spec.params),
		}
		if werr := WriteAudit(stderr, auditFile, rec); werr != nil {
			return UserError(werr, "audit")
		}

		if derr != nil {
			return APIError(derr, spec.action)
		}
		if _, perr := fmt.Fprintln(out, result); perr != nil {
			return UserError(perr, "render")
		}
		return nil
	}
}
