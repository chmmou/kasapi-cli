// Package transport provides the HTTP client that posts SOAP envelopes
// to a KAS endpoint. The client gates outgoing requests behind a
// per-instance "earliest next call" timestamp so callers can honour
// the server-side KasFloodDelay returned by every KasApi response.
//
// Decoding is left to the caller; transport returns the raw response
// body. After parsing a response, the caller reports the new delay
// via Client.RecordDelay so the next call to Do is gated correctly.
//
// Network errors and 5xx responses are retried with exponential
// backoff; 4xx and SOAP-level faults are returned without retry.
//
// See issue #4.
package transport
