package cli

import (
	"errors"
	"fmt"

	"github.com/chmmou/kasapi-cli/internal/api"
	"github.com/chmmou/kasapi-cli/internal/auth"
	"github.com/chmmou/kasapi-cli/internal/config"
	"github.com/chmmou/kasapi-cli/internal/soap"
	"github.com/chmmou/kasapi-cli/internal/transport"
)

// BuildAPIClient resolves credentials from config + env + flags and
// returns an *api.Client wired with a fresh transport and a token
// source matching the requested auth_type.
//
// auth_type=plain  → credentials are passed verbatim to KasApi each
// call via api.StaticTokenSource.
// auth_type=session → the configured AuthData is treated as the
// account password; auth.SessionTokenSource fetches a 40-char session
// token from KasAuth on first use and refreshes it transparently
// after no_auth / unknown_session faults.
//
// Errors are wrapped as *ExitError with ExitUserError so cmd/kasapi-cli
// surfaces them with the expected exit code.
func BuildAPIClient(opts *RootOptions) (*api.Client, error) {
	if opts == nil {
		return nil, UserError(errors.New("nil RootOptions"), "cli")
	}

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

	tr := transport.New()
	ts, err := tokenSource(tr, creds)
	if err != nil {
		return nil, UserError(err, "")
	}
	return api.New(tr, ts), nil
}

func tokenSource(tr *transport.Client, creds config.Credentials) (api.TokenSource, error) {
	switch creds.AuthType {
	case config.AuthPlain:
		return &api.StaticTokenSource{
			Login:    creds.Login,
			AuthData: creds.AuthData,
			AuthType: soap.AuthPlain,
		}, nil
	case config.AuthSession:
		authClient := auth.New(tr, creds.Login, creds.AuthData, soap.AuthPlain, auth.Options{})
		return auth.NewSessionTokenSource(authClient), nil
	default:
		return nil, fmt.Errorf("config: unsupported auth_type %q", creds.AuthType)
	}
}
