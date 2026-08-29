# Units of Work — Price Checker

**Deployment model**: single statically-linked Go binary (`price-checker`). Not microservices — units are **development units** for the per-unit CONSTRUCTION loop. Each unit maps to one or more Go packages.

**Module path**: `github.com/example/price-checker` (placeholder; rename later — Q6=B).

## Code organization strategy (greenfield)

We use the **idiomatic Go layout** rather than the generic `src/{unit-name}/` pattern, because units map cleanly onto Go package directories and Go tooling assumes this structure. Tests live beside the code they test (`*_test.go`), per Go convention, not in a separate `tests/` tree.

```
price-checker/
├── go.mod  go.sum
├── cmd/price-checker/main.go            # U5
├── pkg/schema/                          # U0  (exported contract)
│   ├── breakdown.go  diff.go  usage.go  reason.go  version.go
│   ├── breakdown.schema.json  diff.schema.json  usage.schema.json   (//go:embed)
│   └── *_test.go
├── internal/
│   ├── plan/                            # U1
│   ├── awsauth/                         # U2
│   ├── pricing/                         # U2  (client, cache, catalog, RegionResolver, interfaces)
│   │   └── aws/                         # U2  (16 pricer implementations + fixtures)
│   ├── usage/                           # U3
│   ├── engine/                          # U3  (Aggregator)
│   ├── estimator/                       # U3  (Estimator service)
│   ├── render/                          # U4  (Renderer + DiffRenderer + 4 formats)
│   │   └── html/                        # U4  (embedded template + assets)
│   ├── diff/                            # U4  (Differ service)
│   ├── config/                          # U5
│   └── cli/                             # U5  (App, InputResolver, ExitPolicy, Logging)
├── testdata/                            # shared plan-JSON fixtures
└── .github/workflows/                   # U5  (build/test/release)
```

Per-unit code summaries (markdown) go under `aidlc-docs/construction/{unit-name}/code/`.

---

## Common definition of done (Q4=B), applied per unit

1. Package compiles; all exported interfaces from `components.md` implemented.
2. Example-based unit tests pass.
3. `go vet ./...` and `gofmt -l` clean; `golangci-lint` clean.
4. Property-based tests met for this unit's targets **where the Partial PBT set applies** (PBT-02/03/07/08/09).
5. Recorded-fixture tests for any external-API mapping the unit introduces.
6. Import rules from `component-dependency.md` respected (verified by an `internal/` import-lint test or review).

---

## U0 — `schema`

- **Responsibility**: The public, versioned contract shared by every other unit and by external CI consumers. Serializable types, reason codes, constants, and the authored JSON Schema files.
- **Packages**: `pkg/schema`
- **Components owned**: `Breakdown`, `Resource`, `CostComponent`, `ModuleTotal`, `NotEstimated`, `ReasonCode`, `CoverageSummary`, `RunMetadata`, `DiffResult`, `ResourceDelta`, `UsageFile`, `ResourceUsage`; `SchemaVersion`, `MonthlyHours`; embedded `*.schema.json`; `Validate`, deterministic `MarshalJSON`, usage-file parse/marshal.
- **In-scope stories**: US-MACHINE-OUTPUT-1 (schema + round-trip PBT), foundational for US-MACHINE-OUTPUT-2, US-DIFF-2, US-USAGE-1.
- **Dependencies**: none (stdlib only).
- **Unit DoD additions**: JSON Schema files validate real sample documents; `schema`↔struct drift test; PBT-02 round-trips (JSON marshal/unmarshal; usage YAML) with `rapid`.

## U1 — `plan-ingest`

- **Responsibility**: Turn a `--path`/stdin/Terraform-directory input into the provider-neutral `Plan` model. No pricing, no AWS.
- **Packages**: `internal/plan`
- **Components owned**: PlanLoader, PlanParser, Plan / Resource / AttrMap model + typed getters, `known after apply` detection, change-action classification, module-path + provider(+alias) capture.
- **In-scope stories**: US-LOCAL-ESTIMATE-1, US-LOCAL-ESTIMATE-2.
- **Dependencies**: U0.
- **Unit DoD additions**: fixtures for Terraform 1.x plan JSON (`format_version` 0.1 and 1.x), multi-module, aliased providers, `count`/`for_each`-expanded addresses, delete/replace actions, and a rejected old-format file.

## U2 — `pricing-core`

- **Responsibility**: Everything AWS-pricing: SDK config + credential chain, the shared `PriceListClient` (worker pool + dedup + backoff), the on-disk cache decorator, the `Pricer` interface + `NewCatalog()`, `RegionResolver`, and **all ~16 AWS pricer implementations** (Q5=A).
- **Packages**: `internal/awsauth`, `internal/pricing`, `internal/pricing/aws`
- **Components owned**: AWSConfigProvider, PriceListClient, CachingClient, CacheStore, Pricer, Catalog, RegionResolver; pricers: instance, ebsVolume, ebsSnapshot, eip, rdsInstance, aurora, s3, lambda, lb, natGateway, eksCluster, eksNodeGroup, elastiCache, dynamoDB, cloudWatch, dataTransfer.
- **In-scope stories**: US-PRICING-1, -2, -3, -4, -5 (reason codes for pricing-side failures), -6, -7; US-CACHE-1, -2, -3.
- **Dependencies**: U0, U1 (consumes `plan.Resource`).
- **Unit DoD additions**: recorded Price List `GetProducts` fixtures per pricer; cache concurrency + corruption tests; worker-pool + singleflight dedup test; PBT-02/03 for `PriceDimension` parsing; `internal/pricing/aws` has **no** `aws-sdk-go-v2` import (lint test).

## U3 — `cost-engine`

- **Responsibility**: Combine priced components + usage assumptions into `schema.Breakdown`; own the decimal math and coverage accounting; the `Estimator` orchestration service; the usage-file model and the `usage generate` skeleton.
- **Packages**: `internal/usage`, `internal/engine`, `internal/estimator`
- **Components owned**: UsageModel, UsageSkeletonGenerator, Aggregator (`big.Rat` rollups, 730 h, ordering, coverage summary, metadata), Estimator (`Estimate`, `EstimateBothStates`).
- **In-scope stories**: US-LOCAL-ESTIMATE-3, US-LOCAL-ESTIMATE-4, US-USAGE-1, US-USAGE-2, US-USAGE-3; contributes the aggregation half of US-PRICING-5.
- **Dependencies**: U0, U1, U2.
- **Unit DoD additions**: PBT-03 aggregation invariants (resource total == Σ components; totals ≥ 0; single currency); PBT-02 usage-file round-trip via `internal/usage`; rounding/format policy documented in this unit's Functional Design; deterministic-ordering test.

## U4 — `output-and-diff` (merged; Q2=B)

- **Responsibility**: All presentation — the four renderers — **and** the `Differ` service plus the diff variants of each renderer. Consumes `pkg/schema` only.
- **Packages**: `internal/render`, `internal/render/html`, `internal/diff`
- **Components owned**: Renderer / DiffRenderer interfaces, TableRenderer, JSONRenderer, HTMLRenderer (+ embedded template/CSS/JS), GitHubCommentRenderer, RendererFor factory; Differ; DiffResult construction.
- **In-scope stories**: US-LOCAL-ESTIMATE-3, US-LOCAL-ESTIMATE-4 (component view), US-MACHINE-OUTPUT-1, US-MACHINE-OUTPUT-2 (`--out` writing helper), US-REPORT-1, US-DIFF-1, US-DIFF-2, US-PR-COMMENT-1, US-PR-COMMENT-2.
- **Dependencies**: U0 only.
- **Unit DoD additions**: golden-file tests for every renderer (breakdown + diff); JSON output validated against the embedded schema; HTML report opens from `file://` with zero external requests (asset-inlining test); `NO_COLOR`/non-TTY test; PBT-02 JSON diff round-trip.

## U5 — `cli-app` (with early walking-skeleton increment; Q3=C)

- **Responsibility**: The user-facing program — config resolution with provenance, the `urfave/cli/v2` command tree, input classification, exit-code/threshold policy, stderr logging, `version` metadata, and the release/build pipeline.
- **Packages**: `internal/config`, `internal/cli`, `cmd/price-checker`, `.github/workflows`
- **Components owned**: Config + Resolver, InputResolver, cli.App, ExitPolicy, Logging, main.
- **In-scope stories**: US-CI-GATE-1, US-CI-GATE-2, US-CONFIG-INSTALL-1, US-CONFIG-INSTALL-2, US-CONFIG-INSTALL-3, US-PR-COMMENT-3 (`later` — docs recipe).
- **Dependencies**: U0–U4.
- **Walking-skeleton increment** (built right after U0 + U1): `cmd/price-checker/main.go`, minimal `cli.App` with `version` and a stub `breakdown` that parses a plan and prints resource count. Gives an end-to-end smoke test harness before U2/U3 land. The full unit (config precedence, exit policy, thresholds, all subcommands, packaging) completes last.
- **Unit DoD additions**: exit-code matrix test (0/1/2/3 + precedence); config-precedence test (`flag > env > file > default`) with provenance assertions; cross-platform `go build` for the 5 targets in CI; `version` works with no network.

---

## Boundary validation

- Every unit's packages fall within one region of the import graph in `component-dependency.md`; no unit spans a forbidden edge.
- `internal/render` + `internal/diff` (U4) import only `pkg/schema` (U0) — satisfies NFR-5.3.
- Only U2 packages `internal/awsauth` + `internal/pricing` import the AWS SDK; `internal/pricing/aws` does not.
- No cyclic unit dependency: U0 ← U1 ← U2 ← U3 ← U4 ← U5 is a chain (U4 depends on U0 only; U5 on all).
- All 30 stories assigned (see `unit-of-work-story-map.md`); 1 story (`US-PR-COMMENT-3`) is `later`-tagged and lands as docs in U5.
