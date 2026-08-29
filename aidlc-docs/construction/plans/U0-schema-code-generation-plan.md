# U0 `schema` — Code Generation Plan (consolidated cadence)

Single source of truth for U0 code generation. Cadence: design + code in one gate.
Code location: `pkg/schema/` (workspace root, never `aidlc-docs/`). Docs: `aidlc-docs/construction/U0-schema/code/`.

Stories: US-MACHINE-OUTPUT-1 (owned).

| # | Step | Status |
|---|---|---|
| 1 | `go.mod` (module `github.com/example/price-checker`, `go 1.23`, deps) | [x] |
| 2 | `doc.go`, `version.go`, `reason.go` | [x] |
| 3 | `money.go` (`StringToRat`, `RatToString`, `MustRat`) | [x] |
| 4 | `breakdown.go` (types + `NewBreakdown` + `Canonicalize`) | [x] |
| 5 | `diff.go` (types + `NewDiffResult` + `Canonicalize`) | [x] |
| 6 | `usage.go` (`UsageFile`, parse/marshal, JSON marshalers) | [x] |
| 7 | `*.schema.json` (breakdown, diff, usage) + `schemas.go` embed | [x] |
| 8 | `schematest/schematest.go` (exported sample builders) | [x] |
| 9 | Tests: money, reason, schema-validation, round-trip PBT, canonicalize, import-lint | [x] |
| 10 | `aidlc-docs/construction/U0-schema/code/code-summary.md` | [x] |

Build/lint/test execution deferred to Build and Test stage (Go toolchain not present in this environment).
