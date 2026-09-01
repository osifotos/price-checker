# U0 `schema` — Consolidated Design (Functional + NFR Requirements + NFR Design)

**Unit**: U0 `schema` · **Package**: `pkg/schema` · **Depends on**: nothing (stdlib only)
**Owns stories**: US-MACHINE-OUTPUT-1 · **Supports**: US-LOCAL-ESTIMATE-1, US-PRICING-1/5/7, US-USAGE-1, US-MACHINE-OUTPUT-2, US-DIFF-2

---

## 1. Functional Design

### 1.1 Domain entities

| Entity | Fields (JSON name) | Notes |
|---|---|---|
| `Breakdown` | `schema_version`, `currency`, `period`, `resources[]`, `modules[]`, `total_monthly`, `total_hourly`, `not_estimated[]`, `summary`, `metadata` | Root document for `--format json` |
| `Resource` | `address`, `type`, `name`, `module`, `region`, `monthly_cost`, `hourly_cost`, `cost_components[]` | One planned resource |
| `CostComponent` | `name`, `unit`, `price_per_unit`, `monthly_quantity`, `monthly_cost`, `hourly_cost`, `usage_based` (bool) | One priced line item |
| `ModuleTotal` | `module`, `monthly_cost`, `hourly_cost` | Rollup per module path (`""` = root) |
| `NotEstimated` | `address`, `type`, `component` (opt), `reason_code`, `message` | Coverage gap |
| `CoverageSummary` | `resources_total`, `resources_estimated`, `resources_not_estimated`, `components_estimated`, `components_not_estimated` | |
| `RunMetadata` | `tool_version`, `generated_at` (RFC3339 UTC), `plan_format_version`, `regions[]`, `usage_file_used` (bool), `price_queries`, `cache_hit_ratio`, `dedup_hits` | |
| `DiffResult` | `schema_version`, `currency`, `period`, `changes[]`, `total_prior_monthly`, `total_new_monthly`, `total_delta_monthly`, `total_delta_hourly`, `summary`, `metadata` | Root for diff output |
| `ResourceDelta` | `address`, `type`, `kind` (`added`/`removed`/`changed`/`unchanged`), `prior_monthly`, `new_monthly`, `delta_monthly`, `delta_hourly`, `not_estimated` (bool) | |
| `UsageFile` | `version` (string), `resource_usage` (map address → `ResourceUsage`) | Infracost-style input |
| `ResourceUsage` | `map[string]float64` (component key → monthly quantity) | Free-form, validated against catalog by U3 |

### 1.2 Money representation

- **Wire form**: decimal **string** (e.g. `"12.4830"`), never a JSON number — avoids float precision loss for consumers.
- **In-Go form for callers**: `*big.Rat`. `pkg/schema` provides `Money` helpers:
  - `RatToString(r *big.Rat, dp int) string` — fixed decimal places, half-up rounding.
  - `StringToRat(s string) (*big.Rat, error)` — exact parse.
  - Struct fields are stored as `string`; a `MoneyRat(field string) (*big.Rat, error)` convenience is provided.
- Display precision: `price_per_unit` 10 dp; costs 4 dp internally in JSON, renderers round to 2 dp for humans (renderer concern, U4).

### 1.3 Determinism (US-MACHINE-OUTPUT-1)

- `Breakdown` and `DiffResult` implement `json.Marshaler` with:
  - object keys emitted in a **fixed documented order** (struct field order, hand-written encoder or `encoding/json` with ordered struct — Go's `encoding/json` already emits struct fields in declaration order, so a plain struct with no `map` at top level is deterministic).
  - the only maps (`ResourceUsage`, `UsageFile.ResourceUsage`) are encoded with **sorted keys** via a custom `MarshalJSON`.
  - slices are pre-sorted by the producer (U3/U4); `schema` documents the required sort key but does not re-sort (keeps it a pure DTO). A `Canonicalize()` method is provided that sorts all slices by their documented key, for producers to call before marshalling.
- `generated_at` is injected by the caller (so tests can pin it); `schema` does not call `time.Now()`.

### 1.4 Reason codes (closed enum)

`UNSUPPORTED_TYPE`, `MISSING_ATTRIBUTE`, `UNKNOWN_AFTER_APPLY`, `NO_USAGE_DATA`, `PRICING_API_ERROR`, `UNSUPPORTED_CONFIGURATION`.
`ReasonCode` is a `string` type with these constants + `Valid()` + `Message()` (default human text).

### 1.5 Versioning

- `SchemaVersion = "1.0"`. Additive changes keep it; any removal/rename/semantic change bumps the **minor** for additive-optional and **major** for breaking. Documented in `pkg/schema/doc.go` and README.
- `MonthlyHours = 730` (constant, used by producers).

### 1.6 Public functions

| Function | Purpose |
|---|---|
| `ParseUsageFile(b []byte) (UsageFile, error)` | Strict YAML parse; unknown top-level keys → error; unknown per-resource component keys are **kept** (U3 warns, not schema). |
| `(UsageFile) Marshal() ([]byte, error)` | Deterministic YAML (sorted addresses + keys) — round-trip target for PBT-02. |
| `(UsageFile) MarshalJSON` / `UnmarshalJSON` | JSON round-trip (PBT-02). |
| `(*Breakdown) Canonicalize()` / `(*DiffResult) Canonicalize()` | Sort all slices by documented key in place. |
| `NewBreakdown(...)`, `NewDiffResult(...)` | Constructors that set `schema_version`. |
| `BreakdownSchemaJSON`, `DiffSchemaJSON`, `UsageSchemaJSON` | `[]byte`, `//go:embed`-ed JSON Schema (Draft 2020-12). |

### 1.7 Error handling

- Parse/validate functions return wrapped errors (`fmt.Errorf("schema: ...: %w", err)`).
- No panics on malformed input.
- `StringToRat` rejects `NaN`, `Inf`, empty, and non-decimal.

### 1.8 Business scenarios / edge cases

- Empty plan → valid `Breakdown` with zero totals, empty slices (not null), `summary` all zero.
- All resources not-estimated → totals `"0"`, `not_estimated` populated, `summary.resources_estimated == 0`.
- Negative cost → rejected by `StringToRat` caller contract? No — allowed to represent credits later; schema permits `-` but v1 producers never emit it (documented). `Canonicalize` does not clamp.
- Very large plan (10k resources) → no per-item allocation surprises; slices sized once.
- Unicode in resource names/addresses → preserved as UTF-8; JSON encoder escapes correctly.

---

## 2. NFR Requirements (unit-scoped)

| NFR | This unit's obligation |
|---|---|
| NFR-2.1 correctness | Decimal-string money on the wire; `big.Rat` helpers; no float in cost math paths |
| NFR-2.3 PBT (Partial) | PBT-02 round-trips: `Breakdown` JSON, `DiffResult` JSON, `UsageFile` YAML + JSON. PBT-07 generators: domain generators for `Breakdown`/`UsageFile`. PBT-08: `rapid` with logged seed. PBT-09: framework = `pgregory.net/rapid` |
| NFR-3.2 privacy | Types carry only cost-relevant identifiers (address, type, region, component name) — **no arbitrary attribute values**. `ResourceUsage` holds numbers only |
| NFR-5.3 isolation | `pkg/schema` imports stdlib + `yaml.v3` only; nothing internal; safe for external consumers to import |
| NFR-6 portability | Pure Go, no cgo, no build tags |
| NFR-7.2 observability | `RunMetadata` fields for `price_queries`, `cache_hit_ratio`, `dedup_hits` |

### 2.1 Dependencies (from Application Design §6, accepted)

| Purpose | Import | Scope |
|---|---|---|
| YAML parse/emit | `gopkg.in/yaml.v3` | non-test |
| PBT | `pgregory.net/rapid` | test |
| JSON Schema validation | `github.com/santhosh-tekuri/jsonschema/v5` | test |

`go` directive: `go 1.23`. Module: `github.com/mtosin123/tf-price_checker`.

---

## 3. NFR Design (patterns)

- **DTO purity**: `pkg/schema` has no I/O, no clock, no env access. Callers inject `generated_at`. This makes every function pure and trivially testable.
- **Custom `MarshalJSON` only where needed**: top-level docs rely on struct field order; only the two map-bearing types get hand-written marshalers with sorted keys. Keeps the surface small.
- **Embedded schemas as the contract**: `//go:embed *.schema.json`; a drift test (`TestStructsMatchSchema`) marshals a fully-populated fixture and validates it against the embedded schema, failing if a field is added to a struct without updating the schema (and a hand-maintained field list check for the reverse).
- **`Canonicalize` instead of "always sorted"**: producers call it once before output; avoids repeated sorts and keeps `schema` from needing comparison logic for partially-built values.
- **Golden fixtures** live in `pkg/schema/testdata/` (`breakdown_full.json`, `diff_full.json`, `usage_full.yaml`) and are reused by downstream units' tests.

### 3.1 File plan (`pkg/schema/`)

| File | Contents |
|---|---|
| `doc.go` | Package doc, versioning policy |
| `version.go` | `SchemaVersion`, `MonthlyHours` |
| `reason.go` | `ReasonCode` + constants + `Valid()` / `Message()` |
| `money.go` | `RatToString`, `StringToRat`, rounding helper |
| `breakdown.go` | `Breakdown`, `Resource`, `CostComponent`, `ModuleTotal`, `NotEstimated`, `CoverageSummary`, `RunMetadata`, `NewBreakdown`, `Canonicalize` |
| `diff.go` | `DiffResult`, `ResourceDelta`, `DeltaKind` + consts, `NewDiffResult`, `Canonicalize` |
| `usage.go` | `UsageFile`, `ResourceUsage`, `ParseUsageFile`, `Marshal`, custom JSON marshalers |
| `schemas.go` | `//go:embed` the three `*.schema.json` as exported `[]byte` |
| `breakdown.schema.json`, `diff.schema.json`, `usage.schema.json` | JSON Schema Draft 2020-12 |
| `testdata/*` | golden fixtures |
| `*_test.go` | example-based + `rapid` PBT + schema-drift + golden tests |

### 3.2 Test targets (DoD)

- `TestReasonCodeValid`, `TestMoneyRoundTrip` (example + `rapid`), `TestStringToRatRejects`.
- `rapid` `TestBreakdownJSONRoundTrip`, `TestDiffJSONRoundTrip`, `TestUsageFileRoundTrip` (YAML + JSON) — seed logged.
- `TestBreakdownMatchesSchema`, `TestDiffMatchesSchema`, `TestUsageMatchesSchema` (validate golden fixture against embedded schema).
- `TestCanonicalizeSortsDeterministically`.
- `TestEmptyBreakdownEncodesEmptyArraysNotNull`.
- import-lint: `TestNoDisallowedImports` (only stdlib + `yaml.v3`).

---

## 4. Open items → later units

- Exact `ResourceUsage` key vocabulary → U3 (validated against `pricing.Catalog`).
- Whether renderers need extra display-only fields → U4 may request additive fields (minor version bump).
