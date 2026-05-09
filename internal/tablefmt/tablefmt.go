// Package tablefmt holds presentation-shaped constants and helpers
// shared across the read-domain modules. It is a leaf package with
// no domain dependencies, so each internal/<module> can import it
// without inverting the cli/<-domain layering.
package tablefmt

// FieldValueHeaders is the canonical column pair returned by
// single-record TableHeaders() implementations. The list view of a
// resource carries per-field columns (LOGIN, NAME, ...); the
// singular view collapses to a key/value list rendered with these
// two fixed headers.
//
// Centralising the literal here means a future rename ("KEY"/"VALUE",
// localisation, ...) is one diff rather than thirteen.
var FieldValueHeaders = []string{"FIELD", "VALUE"}
