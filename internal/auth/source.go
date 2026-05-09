package auth

import (
	"context"
	"sync"
	"time"

	"github.com/chmmou/kasapi-cli/internal/session"
	"github.com/chmmou/kasapi-cli/internal/soap"
)

// SessionTokenSource caches a credential token fetched via Client and
// re-fetches it on demand. It implements api.TokenSource.
//
// When Store is set, the cache is also persisted to disk so a token
// obtained interactively (with --otp) survives across CLI invocations
// for as long as ExpiresAt has not been reached. Lifetime mirrors the
// server-side session_lifetime; a zero Lifetime falls back to
// session.DefaultLifetime so the local view does not grow stale
// faster than the server.
//
// UpdateLifetime mirrors session_update_lifetime=Y on the server: when
// true, Heartbeat (called by api.Client after every successful call)
// extends ExpiresAt to Now+Lifetime so the cache tracks the rolling
// window the server applies.
//
// SessionTokenSource is safe for concurrent use.
type SessionTokenSource struct {
	Client         *Client
	Store          *session.Store
	Lifetime       time.Duration
	UpdateLifetime bool
	Now            func() time.Time

	mu        sync.Mutex
	loaded    bool
	token     string
	expiresAt time.Time

	// Snapshot of the user-configured Lifetime / UpdateLifetime captured
	// on first Credentials call. Loading a persisted session adopts the
	// values it was created with so Heartbeats stay consistent for the
	// rest of that session's life. Invalidate then restores these
	// snapshot values so the *next* fresh session honours the CLI flags
	// of the current run, not the stale persisted properties.
	configCaptured           bool
	configuredLifetime       time.Duration
	configuredUpdateLifetime bool
}

// NewSessionTokenSource returns a source that lazily fetches the token
// on first use and does not persist it. Set Store, Lifetime, and
// UpdateLifetime on the returned value to enable cross-invocation
// caching.
func NewSessionTokenSource(c *Client) *SessionTokenSource {
	return &SessionTokenSource{Client: c}
}

// Credentials returns the cached token. Order of preference:
//  1. an in-memory token that has not expired,
//  2. a persisted token loaded from Store that has not expired,
//  3. a fresh token from KasAuth (also persisted when Store is set).
func (s *SessionTokenSource) Credentials(ctx context.Context) (string, string, soap.AuthType, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.configCaptured {
		s.configuredLifetime = s.Lifetime
		s.configuredUpdateLifetime = s.UpdateLifetime
		s.configCaptured = true
	}

	if s.cachedValid() {
		return s.Client.Login, s.token, soap.AuthSession, nil
	}
	if !s.loaded && s.Store != nil {
		s.loaded = true
		if e, err := s.Store.Load(s.Client.Login); err == nil && e != nil {
			s.token = e.Token
			s.expiresAt = e.ExpiresAt
			if s.cachedValid() {
				// Adopt the server-side properties of this session so
				// later Heartbeats use the lifetime the session was
				// created with, not whatever flags the current CLI run
				// happens to carry. session_update_lifetime is a
				// KasAuth-time decision, not a runtime knob.
				if e.LifetimeSeconds > 0 {
					s.Lifetime = time.Duration(e.LifetimeSeconds) * time.Second
				}
				s.UpdateLifetime = e.UpdateLifetime
				return s.Client.Login, s.token, soap.AuthSession, nil
			}
			s.token = ""
			s.expiresAt = time.Time{}
		}
	}
	t, err := s.Client.GetCredentialToken(ctx)
	if err != nil {
		return "", "", "", err
	}
	s.token = t
	s.expiresAt = s.now().Add(s.lifetime())
	if s.Store != nil {
		entry := session.Entry{
			Token:           t,
			ExpiresAt:       s.expiresAt,
			LifetimeSeconds: int(s.lifetime() / time.Second),
			UpdateLifetime:  s.UpdateLifetime,
		}
		_ = s.Store.Save(s.Client.Login, entry)
	}
	return s.Client.Login, s.token, soap.AuthSession, nil
}

// Invalidate clears the cached token in memory and on disk so the next
// Credentials call re-authenticates against KasAuth. If the source has
// adopted a persisted session's Lifetime / UpdateLifetime, those are
// reset to the user-configured values so the fresh session created by
// the next Credentials call respects the current CLI flags rather than
// the now-discarded session's properties.
func (s *SessionTokenSource) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = ""
	s.expiresAt = time.Time{}
	s.loaded = true
	if s.configCaptured {
		s.Lifetime = s.configuredLifetime
		s.UpdateLifetime = s.configuredUpdateLifetime
	}
	if s.Store != nil {
		_ = s.Store.Delete(s.Client.Login)
	}
}

// Heartbeat extends the cached expiry by Lifetime when UpdateLifetime
// is true, mirroring the server's session_update_lifetime=Y rolling
// window. It is a no-op when UpdateLifetime is false or no token is
// cached; when a Store is wired up the refreshed expiry is also
// persisted, otherwise the rolling window stays in-memory only.
// Called by api.Client after every successful KasApi call.
func (s *SessionTokenSource) Heartbeat() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.UpdateLifetime || s.token == "" {
		return
	}
	s.expiresAt = s.now().Add(s.lifetime())
	if s.Store != nil {
		entry := session.Entry{
			Token:           s.token,
			ExpiresAt:       s.expiresAt,
			LifetimeSeconds: int(s.lifetime() / time.Second),
			UpdateLifetime:  true,
		}
		_ = s.Store.Save(s.Client.Login, entry)
	}
}

func (s *SessionTokenSource) cachedValid() bool {
	if s.token == "" {
		return false
	}
	if s.expiresAt.IsZero() {
		return true
	}
	return s.now().Before(s.expiresAt)
}

func (s *SessionTokenSource) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *SessionTokenSource) lifetime() time.Duration {
	if s.Lifetime > 0 {
		return s.Lifetime
	}
	return session.DefaultLifetime
}
