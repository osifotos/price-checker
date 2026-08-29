package schema

import _ "embed"

// BreakdownSchemaJSON is the JSON Schema (Draft 2020-12) for the Breakdown
// document. It is the consumer-facing contract; tests validate real output
// against it.
//
//go:embed breakdown.schema.json
var BreakdownSchemaJSON []byte

// DiffSchemaJSON is the JSON Schema for the DiffResult document.
//
//go:embed diff.schema.json
var DiffSchemaJSON []byte

// UsageSchemaJSON is the JSON Schema for the usage file (JSON form).
//
//go:embed usage.schema.json
var UsageSchemaJSON []byte
