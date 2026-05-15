// Package session covers the KAS API session domain in two cohesive
// halves:
//
//   - Store persists KasAuth credential tokens between CLI invocations
//     (a single sessions.toml keyed by login, written 0600 next to the
//     config file) so an interactive login including 2FA is not
//     repeated while the server keeps the session alive.
//   - Client wraps delete_session, the session-write endpoint that is
//     genuinely distinct from the KasAuth flow: it invalidates a
//     server-side session identified by the (login, token) auth tuple.
//
// add_session is not a distinct endpoint — it is the KasAuth
// credential-token flow itself (same parameters, KasAuth.php envelope,
// bare-string token reply) and is implemented in internal/auth
// (Client.GetCredentialToken / SessionTokenSource). See issues #5,
// #11, and #60.
package session
