# U0 `schema` — Code Summary

**Location**: `pkg/schema/` (+ `pkg/schema/schematest/`) · **Module**: `github.com/example/price-checker` (`go.mod` created)

> Note: the Go toolchain is not installed in this environment, so `go build` / `go test` / `gofmt` have not been run here. They run in the Build and Test stage. `go mod tidy` must be run there to populate `go.sum`.

## Files created

| File | Contents |
|---|---|
| `go.mod` | module + `go 1.23`; requires `jsonschema/v5` (test), `yaml.v3`, `rapid` (test) |
| `pkg/schema/doc.go` | package doc: money, determinism, versioning policy |
| `pkg/schema/version.go` | `SchemaVersion="1.0"`, `MonthlyHours=730`, `Period` (`month`/`hour`) + `Valid()` |
| `pkg/schema/reason.go` | `ReasonCode` + 6 constants, `Valid()`, `Message()`, `ReasonCodes()` (sorted) |
| `pkg/schema/money.go` | `StringToRat` (exact, rejects sci-notation / `+` / junk), `RatToString` (fixed dp, half-away-from-zero), `MustRat` |
| `pkg/schema/breakdown.go` | `Breakdown`, `Resource`, `CostComponent`, `ModuleTotal`, `NotEstimated`, `CoverageSummary`, `RunMetadata`; `NewBreakdown`, `Canonicalize` (sorts slices, nil→[], sorts regions) |
| `pkg/schema/diff.go` | `DiffResult`, `ResourceDelta`, `DeltaKind` + consts + `Valid()`; `NewDiffResult`, `Canonicalize` (order by descending |delta|, then address) |
| `pkg/schema/usage.go` | `UsageFile`, `ResourceUsage`; `ParseUsageFile` (strict, `KnownFields`), `Marshal` (deterministic YAML), `MarshalJSON`/`UnmarshalJSON` (sorted-key JSON) |
| `pkg/schema/schemas.go` | `//go:embed` → `BreakdownSchemaJSON`, `DiffSchemaJSON`, `UsageSchemaJSON` |
| `pkg/schema/breakdown.schema.json` | JSON Schema Draft 2020-12, `additionalProperties:false`, decimal-string pattern, reason-code enum |
| `pkg/schema/diff.schema.json` | JSON Schema for diff output |
| `pkg/schema/usage.schema.json` | JSON Schema for the usage file (JSON form) |
| `pkg/schema/schematest/schematest.go` | exported `SampleBreakdown()`, `SampleDiff()`, `SampleUsage()` for downstream units' tests |

## Tests created (`package schema_test`, external)

| File | Covers | DoD item |
|---|---|---|
| `money_test.go` | `RatToString` cases, `StringToRat` rejections, **PBT-02** format↔parse round trip (`rapid`) | example + PBT |
| `reason_test.go` | all reason codes valid + have messages + sorted; unknown rejected | example |
| `schema_test.go` | sample + empty `Breakdown`, `DiffResult`, `UsageFile` **validate against embedded JSON Schema**; empty slices encode as `[]` not `null` | recorded-fixture / contract |
| `roundtrip_test.go` | **PBT-02** JSON round trip for `Breakdown` & `DiffResult`; YAML + JSON round trip for `UsageFile`; domain generators (**PBT-07**) for addresses / decimals / documents | PBT |
| `canonicalize_test.go` | `Canonicalize` is order-independent; diff orders by |delta| | example |
| `imports_test.go` | **NFR-5.3** non-test files import only stdlib + `yaml.v3` | import-lint |

`rapid` runs with a logged seed and shrinking enabled by default (**PBT-08**); framework is `pgregory.net/rapid` (**PBT-09**).

## Stories advanced

- **US-MACHINE-OUTPUT-1** (owned): schema types, `schema_version`, deterministic encoding, embedded JSON Schema, PBT round-trips — implemented at the contract level (JSON renderer that emits it is U4).
- Supports US-LOCAL-ESTIMATE-1, US-PRICING-1/5/7, US-USAGE-1, US-MACHINE-OUTPUT-2, US-DIFF-2 (types now available to downstream units).

## Deviations / notes

- Tests are an **external `schema_test` package** (not internal) so they can share `schematest` fixtures with downstream units without an import cycle; all assertions use exported API only.
- `math/big.Rat` chosen per Q4; `RatToString` rounding = half away from zero (to be reconfirmed as the project-wide policy in U3).
- `ResourceUsage` key vocabulary is intentionally open here; U3 validates keys against `pricing.Catalog`.
