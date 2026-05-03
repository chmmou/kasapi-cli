// Package auth implements the KasAuth.php credential-token flow.
//
// KasAuth differs from KasApi in two ways: the request element is
// <tns:KasAuth> instead of <tns:KasApi>, and the success body is a bare
// xsd:string carrying the 40-character credential token instead of the
// ns2:Map response shape. Faults use the standard SOAP-ENV:Fault element
// and are decoded by the soap package.
//
// Client.GetCredentialToken posts the auth request and returns the
// token. Auth-specific faults surface as *Error with helpers
// IsLoginFailed, IsLoginLocked, and IsOTPPinIncorrect.
//
// SessionTokenSource adapts the client to the api.TokenSource interface,
// caching the token and refreshing it the next time api.Client observes
// an auth failure and calls Invalidate.
//
// See issue #5.
package auth
