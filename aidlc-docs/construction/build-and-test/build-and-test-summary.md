# Build and Test Summary — price-checker

## Verified 2026-08-29 with Go 1.27.0 (darwin/arm64)

`go mod tidy`, `go build ./...`, `go vet ./...`, `gofmt -l .`, and
`go test ./... -count=1` were all run. Results below are **actual, not
projected**. `golangci-lint` and `govulncheck` were not available in the
environment — run them in CI.

### Fixes applied during this pass
| Issue | Fix |
|---|---|
| `internal/plan/imports_test.go`: test type `parser` collided with `go/parser` import | test moved to `package plan_test`, import aliased `goparser` |
| `internal/render/gh.go`: `go vet` "redundant newline" in `Fprintln` | switched to `Fprint` with explicit `\n\n` |
| gofmt: 14 files unformatted | `gofmt -w .` |
| `internal/cli` tests exited the test binary (urfave/cli default `os.Exit` on ExitCoder) | added a no-op `App.ExitErrHandler`; `runWithDeps` maps the exit code |
| `TestComponentInvariants` PBT tolerance too tight for 4dp-rounded hourly on tiny costs | tolerance 0.01 → 0.05 (max rounding error is 0.00005 × 730 ≈ 0.037) |

## Build

| Item | Value |
|---|---|
| Build tool | Go 1.23+ (verified on 1.27.0), modules |
| `go mod tidy` | ✅ ran — `go.sum` written, 14 transitive deps resolved (aws-sdk-go-v2 bumped to v1.32.3, smithy-go, sso/sts, go-md2man) |
| `go build ./...` | ✅ success, no output |
| `go vet ./...` | ✅ clean |
| `gofmt -l .` | ✅ clean (after `gofmt -w .`) |
| Binary smoke test | ✅ `price-checker version`, `usage generate`, `--help`, bad-JSON → exit 1, `--strict` → exit 2, `--format json` valid & schema_version 1.0, region resolution (aliased provider → us-west-2) |
| Artifact | single static binary (`CGO_ENABLED=0`), 5 OS/arch targets via CI |

## Code inventory

| Unit | Packages | Source files | Test files |
|---|---|---|---|
| U0 schema | `pkg/schema` (+`schematest`) | 9 | 6 |
| U1 plan-ingest | `internal/plan` | 5 | 5 |
| U2 pricing-core | `internal/awsauth`, `internal/pricing`, `internal/pricing/aws` | 20 | 8 |
| U3 cost-engine | `internal/usage`, `internal/engine`, `internal/estimator` | 8 | 6 |
| U4 output-and-diff | `internal/render` (+`html`), `internal/diff` | 10 | 4 |
| U5 cli-app | `internal/config`, `internal/cli`, `cmd/price-checker` | 11 | 4 |
| **Total** | 16 packages | ~63 | ~33 |

## Test execution summary (expected)

### Unit tests
- **Command**: `go test ./... -count=1`
- **Result**: ✅ **all 12 test packages pass**, 0 failures (after the 5 fixes above).
- **Included**: example-based across every package + **property-based** (`pgregory.net/rapid`) for money round-trips, plan parsing, cost-component invariants, aggregation invariants, JSON/YAML round-trips, diff delta-sum; **JSON-Schema conformance** of rendered JSON; **import-lint** enforcing the layering (renderers/diff/usage/schema/aws-pricers stay free of the AWS SDK).

### Integration tests
- **Offline cross-unit**: covered by `internal/cli/run_test.go` (end-to-end `Run` with a stubbed querier) + `internal/estimator/estimator_test.go`. Expected: pass.
- **Live AWS Price List**: opt-in (`PRICE_CHECKER_LIVE=1`), **not implemented as code yet** — instructions in `integration-test-instructions.md`. This is where the U2 Price List filter strings get verified and real `GetProducts` fixtures get recorded.
- **Status**: offline expected pass; live not yet exercised.

### Performance tests
- **Targets**: 200-resource plan ≤ 10 s warm / ≤ 60 s cold; warm run issues 0 queries.
- **Status**: not run (needs a built binary + AWS). Design mitigations in place: bounded worker pool (8), singleflight dedup + per-run memo, TTL disk cache.

### Security / hygiene
- `govulncheck ./...`, `go vet ./...`, `golangci-lint run` — **not run in-session**.
- Baseline guarantees (NFR-3) enforced by tests: no-SDK import-lint, cache file perms `0600`, HTML report self-contained + auto-escaped, output limited to `pkg/schema` fields.
- Security extension: **disabled** per Requirements Analysis.

## Known follow-ons (tracked, not blocking the workflow)

1. **Run the toolchain**: `go mod tidy`, `go build`, `go test ./...`, `gofmt -w .`, `golangci-lint run` on a machine with Go 1.23+. Fix any compile errors the static review missed.
2. **Record real Price List fixtures** (U2 DoD item): run the live integration tests once with AWS credentials, persist `GetProducts` responses under `internal/pricing/aws/testdata/`, and tighten any filter strings that miss.
3. **Add a 200-resource plan fixture + generator** for the performance tests.
4. **US-PR-COMMENT-3** (`later`): the GitHub Actions recipe is in `README.md`; wire it as an actual workflow when adopting.

## Overall status

- **Build**: ✅ passes (`go build ./...`, `go vet`, `gofmt`).
- **Unit + offline integration tests**: ✅ all 12 packages pass (`go test ./... -count=1`).
- **Binary smoke tests**: ✅ version / usage generate / help / exit codes / JSON / region resolution.
- **Live pricing accuracy**: ⚠️ **unverified** — no AWS creds in the build environment; pricers return `PRICING_API_ERROR` and the tool degrades gracefully (prices what it can, exits 0). Needs the live integration pass (follow-on 2) to confirm Price List filter strings.
- **golangci-lint / govulncheck**: not run (tools not installed) — run in CI.
- **Ready for Operations**: **Not yet** — pending live pricing verification (follow-on 2).

## Instruction files generated
- `build-instructions.md`
- `unit-test-instructions.md`
- `integration-test-instructions.md`
- `performance-test-instructions.md`
- `security-test-instructions.md`
- `build-and-test-summary.md`
