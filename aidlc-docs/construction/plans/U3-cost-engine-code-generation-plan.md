# U3 `cost-engine` — Code Generation Plan (consolidated cadence)

Code: `internal/usage/`, `internal/engine/`, `internal/estimator/`.
Stories: US-LOCAL-ESTIMATE-3/4, US-USAGE-1/2/3, US-PRICING-5 (aggregation).

| # | Step | Status |
|---|---|---|
| 1 | `internal/usage/keys.go` (keysByType + knownUsageKeys) | [x] |
| 2 | `internal/usage/model.go` (Model, Load, Validate, emptyModel) | [x] |
| 3 | `internal/usage/skeleton.go` (GenerateSkeleton) | [x] |
| 4 | `internal/engine/aggregator.go` (PricedResource, Build) — SDK-free | [x] |
| 5 | `internal/estimator/estimator.go` (Deps, EstimateRequest, Estimate, EstimateBothStates, errgroup fan-out) | [x] |
| 6 | Tests: usage (model, skeleton, import-lint), engine (basic, empty, determinism, PBT), estimator (happy/unsupported/region/error/determinism/both-states/metadata) | [x] |
| 7 | `code-summary.md` | [x] |

Build/lint/test deferred to Build and Test.
