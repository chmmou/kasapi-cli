package auth

import (
	"context"
	"sync"

	"github.com/chmmou/kasapi-cli/internal/soap"
)

// SessionTokenSource caches a credential token fetched via Client and
// re-fetches it on demand. It implements api.TokenSource: api.Client
// calls Credentials before every request and Invalidate after an auth
// failure, which causes the next Credentials call to obtain a fresh
// token via the underlying KasAuth client.
//
// SessionTokenSource is safe for concurrent use.
type SessionTokenSource struct {
	Client *Client

	mu    sync.Mutex
	token string
}

// NewSessionTokenSource returns a source that lazily fetches the token
// on first use. The login it reports to api.Client is the one stored on
// the underlying KasAuth client; the auth_type for KasApi calls is
// always session, since the credential token is a session token.
func NewSessionTokenSource(c *Client) *SessionTokenSource {
	return &SessionTokenSource{Client: c}
}

// Credentials returns the cached token, fetching a new one if the
// cache is empty.
func (s *SessionTokenSource) Credentials(ctx context.Context) (string, string, soap.AuthType, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == "" {
		t, err := s.Client.GetCredentialToken(ctx)
		if err != nil {
			return "", "", "", err
		}
		s.token = t
	}
	return s.Client.Login, s.token, soap.AuthSession, nil
}

// Invalidate clears the cached token so the next Credentials call
// re-authenticates against KasAuth.
func (s *SessionTokenSource) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = ""
}
