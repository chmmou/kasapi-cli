package auth

import (
	"context"
	"log/slog"
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
// Loading a persisted entry temporarily overrides Lifetime and
// UpdateLifetime with the values the session was created with so
// Heartbeats stay consistent. The user-supplied values are snapshotted
// on first Credentials call and restored by Invalidate so the next
// fresh session honours the current run's CLI flags rather than the
// discarded session's properties.
//
// SessionTokenSource is safe for concurrent use.
type SessionTokenSource struct {
	Client         *Client
	Store          *session.Store
	Lifetime       time.Duration
	UpdateLifetime bool
	Now            func() time.Time

	// Logger receives Warn events when loading, persisting, or deleting
	// the session cache on disk fails. The in-memory cache still works
	// and the next invocation simply re-bootstraps via KasAuth, but
	// surfacing disk-full / permission errors helps debug field issues.
	// Nil is treated as a discard logger.
	Logger *slog.Logger

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
// caching. Logger is seeded with the package discard logger so callers
// may write to it unconditionally; replace it (e.g. with a stderr
// handler) to surface persist/delete failures from the field.
func NewSessionTokenSource(c *Client) *SessionTokenSource {
	return &SessionTokenSource{Client: c, Logger: discardLogger}
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
		e, err := s.Store.Load(ctx, s.Client.Login)
		switch {
		case err != nil:
			// A corrupt or unreadable sessions.toml must not be silent:
			// the in-memory cache still works and we re-bootstrap via
			// KasAuth below, but surfacing the load failure mirrors the
			// Save / Delete / Heartbeat paths and helps debug field
			// issues instead of masking a stale-cache root cause.
			s.logger().Warn("auth: session store load failed; bootstrapping a fresh token via KasAuth", "err", err)
		case e != nil:
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
		if err := s.Store.Save(ctx, s.Client.Login, entry); err != nil {
			s.logger().Warn("auth: session store save failed; cache stays in-memory", "err", err)
		}
	}
	return s.Client.Login, s.token, soap.AuthSession, nil
}

// Invalidate clears the cached token in memory and on disk so the next
// Credentials call re-authenticates against KasAuth. If the source has
// adopted a persisted session's Lifetime / UpdateLifetime, those are
// reset to the user-configured values so the fresh session created by
// the next Credentials call respects the current CLI flags rather than
// the now-discarded session's properties.
//
// Invalidate is called by api.Client on auth-failure paths where the
// caller's ctx may already be cancelled; the disk delete therefore uses
// context.Background so a successful invalidation cannot be lost just
// because the user pressed Ctrl-C between the API failure and the
// cleanup. The in-memory clear happens unconditionally either way.
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
		if err := s.Store.Delete(context.Background(), s.Client.Login); err != nil {
			s.logger().Warn("auth: session store delete failed; in-memory cache cleared", "err", err)
		}
	}
}

// Heartbeat extends the cached expiry by Lifetime when UpdateLifetime
// is true, mirroring the server's session_update_lifetime=Y rolling
// window. It is a no-op when UpdateLifetime is false or no token is
// cached; when a Store is wired up the refreshed expiry is also
// persisted, otherwise the rolling window stays in-memory only.
// Called by api.Client after every successful KasApi call; ctx is the
// call's context so cancellation cuts off the persistence write too.
func (s *SessionTokenSource) Heartbeat(ctx context.Context) {
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
		if err := s.Store.Save(ctx, s.Client.Login, entry); err != nil {
			s.logger().Warn("auth: session store heartbeat save failed; rolling window stays in-memory", "err", err)
		}
	}
}

// logger returns the configured Logger or the package discard logger
// so call sites may invoke s.logger().Warn unconditionally.
func (s *SessionTokenSource) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return discardLogger
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
