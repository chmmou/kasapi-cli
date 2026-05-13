package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/auth"
	"github.com/chmmou/kasapi-cli/internal/config"
	"github.com/chmmou/kasapi-cli/internal/session"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/transport"
)

// BuildAPIClient resolves credentials from config + env + flags and
// returns an *api.Client wired with a fresh transport and a token
// source matching the requested auth_type.
//
// auth_type=plain  → credentials are passed verbatim to KasApi each
// call via api.StaticTokenSource. KasAuth is not contacted; --otp,
// --session-lifetime, and --session-update-lifetime are therefore
// not supported in this mode (these are KasAuth-only parameters).
// auth_type=session → the configured AuthData is treated as the
// account password; auth.SessionTokenSource fetches a 40-char session
// token from KasAuth on first use (forwarding --otp, --session-lifetime,
// and --session-update-lifetime when set) and refreshes it
// transparently after no_auth / unknown_session faults.
//
// Errors are wrapped as *ExitError with ExitUserError so cmd/kasapi-cli
// surfaces them with the expected exit code.
func BuildAPIClient(opts *RootOptions) (*api.Client, error) {
	if opts == nil {
		return nil, UserError(errors.New("nil RootOptions"), "cli")
	}

	logger := buildLogger(opts.Verbose)

	cfg, err := config.Load(opts.ConfigPath)
	if err != nil && !errors.Is(err, config.ErrNoConfig) {
		return nil, UserError(err, "load config")
	}
	creds, err := cfg.Resolve(config.EnvFromOS(), config.Override{
		Profile:  opts.Profile,
		Login:    opts.Login,
		AuthData: opts.AuthData,
		AuthType: opts.AuthType,
	})
	if err != nil {
		return nil, UserError(err, "")
	}
	logger.Info("cli: credentials resolved",
		"login", creds.Login, "auth_type", creds.AuthType, "auth_data", "<redacted>")

	tr := transport.New()
	tr.Logger = logger
	ts, err := tokenSource(tr, creds, sessionOpts{
		ConfigPath:     opts.ConfigPath,
		OTP:            opts.OTP,
		Lifetime:       opts.SessionLifetime,
		UpdateLifetime: opts.SessionUpdateLifetime,
	}, logger)
	if err != nil {
		return nil, UserError(err, "")
	}
	c := api.New(tr, ts)
	c.Logger = logger
	return c, nil
}

// buildLogger returns a stderr text-handler logger when verbose is set,
// otherwise a discard logger. Subcommands obtain a logger via this
// helper so the same instance can be plumbed into transport, api, and
// auth without each package having to consult --verbose itself.
func buildLogger(verbose bool) *slog.Logger {
	if verbose {
		return slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// sessionOpts groups the KasAuth-only flag values plumbed through to
// auth.Options plus the resolved --config path so the session store
// can live next to a custom config file. All fields are optional;
// zero values mean "omit" / "default location".
type sessionOpts struct {
	ConfigPath     string
	OTP            string
	Lifetime       int
	UpdateLifetime string
}

func (s sessionOpts) any() bool {
	return s.OTP != "" || s.Lifetime != 0 || s.UpdateLifetime != ""
}

func tokenSource(tr *transport.Client, creds config.Credentials, s sessionOpts, logger *slog.Logger) (api.TokenSource, error) {
	switch creds.AuthType {
	case config.AuthPlain:
		if s.any() {
			return nil, fmt.Errorf(
				"--otp / --session-lifetime / --session-update-lifetime cannot be used with " +
					"auth_type=plain: these are KasAuth-only parameters and KasAuth is not " +
					"contacted in plain mode (switch to auth_type=session)")
		}
		return &api.StaticTokenSource{
			Login:    creds.Login,
			AuthData: creds.AuthData,
			AuthType: soap.AuthPlain,
		}, nil
	case config.AuthSession:
		authOpts, err := buildAuthOptions(s)
		if err != nil {
			return nil, err
		}
		// KasAuth bootstrap is always plain regardless of session mode; the
		// session token returned by KasAuth is what subsequent KasApi calls use.
		authClient := auth.New(tr, creds.Login, creds.AuthData, soap.AuthPlain, authOpts)
		authClient.Logger = logger
		src := auth.NewSessionTokenSource(authClient)
		src.Logger = logger
		storePath, serr := session.PathFor(s.ConfigPath)
		if serr != nil {
			return nil, fmt.Errorf("session store: %w", serr)
		}
		store, serr := session.New(storePath)
		if serr != nil {
			return nil, fmt.Errorf("session store: %w", serr)
		}
		src.Store = store
		if s.Lifetime > 0 {
			src.Lifetime = time.Duration(s.Lifetime) * time.Second
		}
		if authOpts.UpdateLifetime != nil && *authOpts.UpdateLifetime {
			src.UpdateLifetime = true
		}
		return src, nil
	default:
		return nil, fmt.Errorf("config: unsupported auth_type %q", creds.AuthType)
	}
}

func buildAuthOptions(s sessionOpts) (auth.Options, error) {
	out := auth.Options{OTP: s.OTP}
	if s.Lifetime != 0 {
		if s.Lifetime < 1 || s.Lifetime > 30000 {
			return auth.Options{}, fmt.Errorf("--session-lifetime must be between 1 and 30000 seconds, got %d", s.Lifetime)
		}
		out.Lifetime = s.Lifetime
	}
	switch s.UpdateLifetime {
	case "":
		// omit
	case "Y":
		v := true
		out.UpdateLifetime = &v
	case "N":
		v := false
		out.UpdateLifetime = &v
	default:
		return auth.Options{}, fmt.Errorf("--session-update-lifetime must be 'Y' or 'N', got %q", s.UpdateLifetime)
	}
	return out, nil
}
