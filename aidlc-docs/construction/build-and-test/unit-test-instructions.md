# Unit Test Execution — price-checker

## Run all unit tests
```bash
go test -count=1 ./...
```

`-count=1` disables the test cache so every run is real. Property-based tests
(`pgregory.net/rapid`) log their seed on failure; re-run a specific failure with
`go test -run TestName -rapid.seed=<seed> ./internal/<pkg>`.

## Per-unit test map

| Unit | Package(s) | Key tests |
|---|---|---|
| U0 schema | `pkg/schema`, `pkg/schema/schematest` | money round-trip (PBT), reason codes, **JSON-Schema validation of sample + empty docs**, JSON/YAML round-trips (PBT), canonicalize determinism, import-lint |
| U1 plan-ingest | `internal/plan` | fixture parse, action mapping, module/provider extraction, `format_version` matrix, known-after-apply, loader (stdin/file/dir with fake runner), parse-preserves-addresses (PBT), import-lint |
| U2 pricing-core | `internal/awsauth`, `internal/pricing`, `internal/pricing/aws` | Price List parse, **client dedup (10 concurrent → 1 SDK call)**, pagination, cache miss/hit/TTL/corrupt/perms/concurrency, region resolution, catalog coverage + no-dup, per-pricer via fake `PriceQuerier`, component invariants (PBT), no-SDK import-lint |
| U3 cost-engine | `internal/usage`, `internal/engine`, `internal/estimator` | usage load/validate/skeleton round-trip, aggregation totals + rollups + determinism + invariants (PBT), estimator happy/unsupported/region/error/both-states/metadata |
| U4 output-and-diff | `internal/render`, `internal/diff` | `RendererFor` matrix, table (no ANSI w/ `Color=false`), **JSON validates against embedded schema**, **HTML self-contained** (no external URLs) + XSS-escaped, gh deterministic, diff classification + delta-sum invariant (PBT), import-lint x2 |
| U5 cli-app | `internal/config`, `internal/cli` | config precedence + provenance, exit-code matrix, input classify, **end-to-end `Run`**: version/table/json/`--strict`→2/`--threshold`→3/unknown-format→1/estimator-error→1/`--help` |

## Review results
- **Expected**: all tests pass, 0 failures.
- **Coverage**: run `go test -cover ./...`; the pricing filter-string details are the main uncovered area (see integration notes) — engine/diff/schema/render/config should be high.
- **Report**: `go test -json ./... > test-report.json` for machine consumption.

## Fixing failures
1. Read the failing test's output (file:line + got/want).
2. For a `rapid` failure, copy the printed seed and the shrunk minimal input; add it as a permanent example-based test after fixing.
3. Re-run `go test -run <Test> ./internal/<pkg>` until green, then `go test ./...`.
