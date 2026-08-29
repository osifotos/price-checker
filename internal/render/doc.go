// Package render formats a schema.Breakdown or schema.DiffResult as a table,
// JSON, a self-contained HTML report, or a GitHub PR comment. Renderers depend
// only on pkg/schema: they never read flags, touch the clock, or call os.Exit,
// and they assume the document has already been canonicalized by its producer.
package render
