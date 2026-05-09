// Package api is the generic KasApi.php call surface that all per-resource
// modules build on. It composes the soap codec with the http transport and
// a TokenSource for credentials, feeds KasFloodDelay back to the transport's
// gate after every successful call, and maps SOAP-ENV:Fault bodies into
// typed *Error values whose Code is the stable KAS error string.
//
// Predicate helpers (IsAuthFailure, IsFloodProtection, IsNotFound,
// IsSyntaxError, IsMaxReached, IsInProgress) are provided so callers can
// branch on classes of errors without enumerating individual codes.
//
// On any code classified as an auth failure by IsAuthFailure, Call
// invalidates the TokenSource and retries once with fresh credentials.
// All other faults surface to the caller without retry.
//
// See issue #6.
package api
