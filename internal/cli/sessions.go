package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/config"
	"github.com/chmmou/kasapi-cli/internal/session"
)

// NewSessionsCmd returns the "kasapi-cli sessions" subcommand tree.
//
// Only delete_session has a standalone subcommand: add_session is not a
// distinct endpoint — it is the KasAuth credential-token flow that
// `config init` / auth_type=session already drive transparently (see
// internal/auth). `sessions delete` is the explicit counterpart to the
// implicit logout performed by `config use-profile`.
func NewSessionsCmd(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "Manage KAS session tokens (delete_session)",
		Long: "Manage KAS session tokens.\n\n" +
			"add_session is not a separate endpoint — it is the KasAuth " +
			"credential-token flow driven transparently by auth_type=session " +
			"(see `config init`). Only delete_session is exposed here, as the " +
			"explicit counterpart to the implicit logout in `config use-profile`.",
	}
	cmd.AddCommand(newSessionsDeleteCmd(opts))
	return cmd
}

func newSessionsDeleteCmd(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Invalidate the active profile's cached session token (delete_session)",
		Long: "Invalidate the resolved profile's cached session token, both " +
			"server-side (kas_action=delete_session) and in the local " +
			"sessions.toml cache.\n\n" +
			"Acts on the *currently cached* token only; it never bootstraps a " +
			"fresh token just to delete it. Idempotent: a missing or " +
			"already-invalid session is reported and exits 0. No confirmation " +
			"prompt — deleting a session merely forces a re-authentication on " +
			"the next session-mode call.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger := buildLogger(opts.Verbose)
			storePath, err := session.PathFor(opts.ConfigPath)
			if err != nil {
				return UserError(err, "session store")
			}
			store, err := session.New(storePath)
			if err != nil {
				return UserError(err, "session store")
			}
			revoke := func(ctx context.Context, login, token string) error {
				return revokeSession(ctx, login, token, "", logger)
			}
			return runSessionsDelete(cmd.Context(), opts, revoke, store, logger, cmd.OutOrStdout())
		},
	}
}

// runSessionsDelete logs out the resolved active profile. It looks up
// the profile's *currently cached* session token and, if present,
// invalidates it server-side via revoke (delete_session) and removes
// the on-disk cache entry. The local cache is the authoritative
// client-side state, so its removal is attempted whenever a token was
// found, even if the server-side revoke fails.
//
// Unlike the implicit best-effort revoke in `config use-profile`, this
// explicit command surfaces real failures so scripts notice: any
// transport/KAS error other than an already-invalid token
// (unknown_session, which is idempotent success) returns an API-error
// exit, and a local cache-removal failure is reported truthfully and
// returns a user-error exit (the local cache is a client-side concern,
// like a missing config file) rather than being silently swallowed.
func runSessionsDelete(ctx context.Context, opts *RootOptions, revoke revokeFunc, store *session.Store, logger *slog.Logger, w io.Writer) error {
	creds, err := resolveCreds(opts)
	if err != nil {
		return err
	}
	if creds.AuthType == config.AuthPlain {
		if _, perr := fmt.Fprintf(w, "no session to delete: profile for login %s uses auth_type=plain (KasAuth is never contacted)\n", creds.Login); perr != nil {
			return UserError(perr, "")
		}
		return nil
	}

	entry, err := store.Load(ctx, creds.Login)
	if err != nil {
		return UserError(err, "session store")
	}
	if entry == nil || entry.Token == "" {
		if _, perr := fmt.Fprintf(w, "no active cached session for login %s; nothing to revoke\n", creds.Login); perr != nil {
			return UserError(perr, "")
		}
		return nil
	}

	revokeErr := revoke(ctx, creds.Login, entry.Token)
	derr := store.Delete(ctx, creds.Login)
	if derr != nil {
		logger.Warn("sessions delete: session store delete failed",
			"login", creds.Login, "err", derr)
	}

	// The local cache message must reflect whether store.Delete actually
	// succeeded — never claim it was cleared when it was not.
	cacheNote := "Cleared the local cache."
	if derr != nil {
		cacheNote = "The local cache could NOT be cleared (see --verbose)."
	}

	switch {
	case revokeErr == nil:
		if _, perr := fmt.Fprintf(w, "Deleted server-side session for login %s. %s\n", creds.Login, cacheNote); perr != nil {
			return UserError(perr, "")
		}
		if derr != nil {
			return UserError(derr, "session store delete")
		}
		return nil
	case api.IsCode(revokeErr, api.CodeUnknownSession):
		logger.Info("sessions delete: token already invalid server-side",
			"login", creds.Login)
		if _, perr := fmt.Fprintf(w, "Session for login %s was already invalid server-side. %s\n", creds.Login, cacheNote); perr != nil {
			return UserError(perr, "")
		}
		if derr != nil {
			return UserError(derr, "session store delete")
		}
		return nil
	default:
		logger.Warn("sessions delete: server-side revoke failed",
			"login", creds.Login, "err", revokeErr)
		if _, perr := fmt.Fprintf(w, "Server-side delete_session for login %s failed. %s\n", creds.Login, cacheNote); perr != nil {
			return UserError(perr, "")
		}
		return APIError(revokeErr, "delete_session")
	}
}
