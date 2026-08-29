// Package schema defines the public, versioned data contract for price-checker.
//
// Everything a CI consumer of `price-checker --format json` depends on lives
// here: the Breakdown and DiffResult documents, the closed set of NotEstimated
// reason codes, the usage-file types, and the embedded JSON Schema documents
// that describe them.
//
// # Money
//
// Monetary values cross the wire as decimal strings (for example "12.4830"),
// never as JSON numbers, so consumers never suffer binary-float rounding. Use
// StringToRat / RatToString to convert to and from *big.Rat for arithmetic.
//
// # Determinism
//
// The document types are plain structs whose fields encode in declaration
// order. The only maps (ResourceUsage) use key-sorted marshalers. Producers
// call Canonicalize once before encoding to sort every slice by its documented
// key. The generated_at timestamp is supplied by the caller; this package never
// reads the clock, the environment, or the filesystem.
//
// # Versioning
//
// SchemaVersion is "1.0". Additive, optional fields keep the major version and
// bump the minor. Any removal, rename, or semantic change to an existing field
// bumps the major version.
package schema
