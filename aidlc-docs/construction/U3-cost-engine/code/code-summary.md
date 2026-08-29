# U3 `cost-engine` — Code Summary

**Locations**: `internal/usage/`, `internal/engine/`, `internal/estimator/`

> Go toolchain absent; build/lint/test run in Build and Test.

## Dependencies

`golang.org/x/sync/errgroup` (already vendored via `golang.org/x/sync`). No other new deps.
`internal/engine` is **SDK-free** (imports only `internal/plan` + `pkg/schema` + stdlib).
`internal/usage` imports only `internal/plan` + `pkg/schema` + stdlib (lint-enforced — no `internal/pricing`, no SDK).

## Files

### `internal/usage/`
| File | Contents |
|---|---|
| `model.go` | `Model` interface (`For`/`Warnings`/`Used`/`Validate`); `Load(path)` (empty path → `emptyModel`; malformed YAML → error; unknown keys → warnings); `Validate(planAddrs)` warns on file addresses not in the plan |
| `keys.go` | `keysByType` (usage-based inputs per resource type, with descriptions) + flattened `knownUsageKeys` |
| `skeleton.go` | `GenerateSkeleton(plan, w)` — commented YAML with zero values for every in-plan resource that has usage-based components; valid input to `Load` |

### `internal/engine/`
| File | Contents |
|---|---|
| `aggregator.go` | `PricedResource`, `Aggregator.Build(priced, meta, period)` — per-resource / per-module / project roll-ups (`big.Rat`, 4dp), `CoverageSummary`, region set, `Canonicalize`; empty input → valid zero breakdown |

### `internal/estimator/`
| File | Contents |
|---|---|
| `estimator.go` | `Deps`, `EstimateRequest`, `Estimator`; `Estimate` (load → parse → usage load+validate → bounded `errgroup` fan-out of pricers → `Aggregator.Build`); `EstimateBothStates` (prior + planned from one plan, shared warm-cache querier); `metadata` pulls `QueryStats` via a local `interface{ Stats() }` assertion; `withAttributes` swaps in prior state for the prior pass |

## Error vs not-estimated

- load / parse / malformed usage YAML / ctx cancel → returned `error` (CLI exit 1).
- unresolved region → `MISSING_ATTRIBUTE`; unsupported type → `UNSUPPORTED_TYPE`; pricer fault → `PRICING_API_ERROR`; individual query failures already `NotEstimated` inside the pricer output. Run continues.

## Tests

| File | Covers |
|---|---|
| `usage/model_test.go` | empty path; valid load; unknown key → warning; `Validate` unknown address → warning; malformed YAML → error |
| `usage/skeleton_test.go` | skeleton lists usage keys for in-plan types only, omits keyless types, round-trips through `Load` with no warnings; empty case → `resource_usage: {}` |
| `usage/imports_test.go` | no `internal/pricing` / SDK imports |
| `engine/aggregator_test.go` | basic totals + module rollups + coverage; empty input; **determinism** (byte-identical); **PBT-03** (`rapid`) — project total ≈ Σ resource totals, non-negative, summary counts consistent |
| `estimator/estimator_test.go` | happy + unsupported-type; missing region (+ `--aws-region` override); pricer error → `PRICING_API_ERROR`; load error → exit-1; determinism (5 runs identical); `EstimateBothStates` prior ≠ planned; pinned `GeneratedAt` + stats copied to metadata |

## Stories advanced

- **US-LOCAL-ESTIMATE-3** (breakdown + module totals + coverage footer data), **US-LOCAL-ESTIMATE-4** (per-component data, period, 730h, decimal), **US-USAGE-1** (usage file feeds pricers), **US-USAGE-2** (warnings not failures), **US-USAGE-3** (`GenerateSkeleton`), **US-PRICING-5** (Aggregator collects + counts not-estimated).

## Notes

- `Aggregator.Build` takes `period` explicitly; the CLI sets it from `--period`. Renderers (U4) also honour `--period` independently.
- Prior-state pricing reuses pricers unchanged via `withAttributes`.
