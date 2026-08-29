# U3 `cost-engine` — Consolidated Design

**Unit**: U3 · **Packages**: `internal/usage`, `internal/engine`, `internal/estimator`
**Depends on**: U0 `pkg/schema`, U1 `internal/plan`, U2 `internal/pricing`
**Owns**: US-LOCAL-ESTIMATE-3, US-LOCAL-ESTIMATE-4, US-USAGE-1/2/3, US-PRICING-5 (aggregation half)

---

## 1. Functional Design

### 1.1 `internal/usage`

**UsageModel** — user-supplied monthly usage assumptions.

```go
func Load(path string) (Model, error)          // "" path -> empty model, no error
type Model interface {
    For(address string) schema.ResourceUsage    // zero map if absent
    Warnings() []string
    Used() bool                                 // a non-empty file was loaded
}
```

- Parses via `schema.ParseUsageFile`. Malformed YAML → hard error.
- **Validation (warnings, never fatal)**: unknown component keys (not in `knownUsageKeys`); addresses present in the file but not in the plan (checked by the caller which passes the plan address set to `Validate(addrs)`).
- `knownUsageKeys` is a static set in `keys.go` (`storage_gb`, `monthly_requests`, `request_duration_ms`, `monthly_data_processed_gb`, `snapshot_size_gb`, `monthly_tier1_requests`, `monthly_tier2_requests`, `monthly_lcu`, `monthly_nlcu`, `monthly_glcu`, ...).

**SkeletonGenerator** — `price-checker usage generate`.

```go
func GenerateSkeleton(p *plan.Plan, w io.Writer) error
```

- `usageKeysByType` (static map `resourceType -> []key`) lists the usage-based components each catalogued type exposes.
- Emits YAML: `version: "0.1"` then `resource_usage:` with one block per in-plan resource that has usage-based components, keys with `0` values and an inline `#` comment describing the unit.
- Resources with no usage-based component are omitted.

### 1.2 `internal/engine` — Aggregator

```go
type PricedResource struct {
    Resource     plan.Resource
    Region       string
    Components   []schema.CostComponent
    NotEstimated []schema.NotEstimated
}
type Aggregator struct{}
func (Aggregator) Build(priced []PricedResource, meta schema.RunMetadata) *schema.Breakdown
```

Rules:
- Per resource: `monthly = Σ component.MonthlyCost`, `hourly = Σ component.HourlyCost` (via `pricing.SumMonthly`/`SumHourly` — `big.Rat`).
- `schema.Resource.Module` = `plan.Resource.ModuleAddress`.
- Module totals: group by module address (`""` = root); `resource_count` per module.
- Project `total_monthly` / `total_hourly` = Σ over resources (estimated components only — a not-estimated line contributes nothing).
- `NotEstimated`: concatenate every `PricedResource.NotEstimated`.
- `CoverageSummary`:
  - `resources_total` = len(priced)
  - `resources_estimated` = count with ≥1 component
  - `resources_not_estimated` = count with 0 components
  - `components_estimated` = Σ len(Components)
  - `components_not_estimated` = Σ len(NotEstimated)
- `meta` attached verbatim; `b.Canonicalize()` called last → deterministic.
- Empty input → valid zero breakdown.

### 1.3 `internal/estimator` — Estimator service

```go
type Deps struct {
    Loader      plan.PlanLoader
    Parser      plan.PlanParser
    Catalog     pricing.Catalog
    Querier     pricing.PriceQuerier   // already cache-wrapped by the caller
    Region      pricing.RegionResolver
    Agg         engine.Aggregator
    Log         *slog.Logger
    ToolVersion string
    Now         func() time.Time       // injectable for tests
}
type EstimateRequest struct {
    Source        string
    Stdin         io.Reader
    RegionOverride string
    UsagePath     string
    Concurrency   int                  // resource-level fan-out; default 8
    Period        schema.Period
}
func New(d Deps) *Estimator
func (*Estimator) Estimate(ctx, EstimateRequest) (*schema.Breakdown, error)
func (*Estimator) EstimateBothStates(ctx, EstimateRequest) (prior, planned *schema.Breakdown, error)
```

`Estimate` pipeline:
1. `raw, info := Loader.Load(ctx, req.Source, req.Stdin)` — error → return.
2. `pl := Parser.Parse(raw)` — error → return.
3. `um := usage.Load(req.UsagePath)`; `um.Validate(planAddressSet)`; log warnings.
4. Fan out over `pl.Resources()` with a bounded worker pool (`req.Concurrency`, default 8):
   - `region, ok := Region.Resolve(res, pl.ProviderConfigs, req.RegionOverride)` → if `!ok` record `PricedResource{NotEstimated: [{MISSING_ATTRIBUTE}]}`, continue.
   - `pr, ok := Catalog.For(res.Type)` → if `!ok` record `{NotEstimated: [{UNSUPPORTED_TYPE}]}`, continue.
   - `out, err := pr.Price(ctx, pricing.PricingInput{Resource: res, Usage: um.For(res.Address), Region: region, Q: Querier})`
     - `err != nil` → `{NotEstimated: [{PRICING_API_ERROR}]}` (pricer contract fault; individual query failures are already `NotEstimated` inside `out`).
   - append `PricedResource{res, region, out.Components, out.NotEstimated}`.
5. Build `schema.RunMetadata`: `ToolVersion`, `GeneratedAt = Now().UTC().Format(RFC3339)`, `PlanFormatVersion = pl.FormatVersion`, `Regions = sorted unique`, `UsageFileUsed = um.Used()`, and from `Querier` if it implements `interface{ Stats() pricing.QueryStats }`: `PriceQueries`, `CacheHitRatio`, `DedupHits`.
6. `return Agg.Build(priced, meta), nil`.

`EstimateBothStates`: same, but step 4 runs twice — once over `pl.PriorResources()` reading `PriorAttributes`, once over `pl.Resources()` — reusing the same `Querier` (cache warm on the second pass). Returns `(priorBreakdown, plannedBreakdown)`.

For the prior pass, a synthetic `plan.Resource` is built with `Attributes = res.PriorAttributes` so pricers work unchanged.

### 1.4 Error vs not-estimated (unchanged from services.md)

| Case | Handling |
|---|---|
| load / parse / usage-YAML-malformed / ctx cancelled | return `error` (→ CLI exit 1) |
| region unresolved / unsupported type / pricer fault / individual query failure | `schema.NotEstimated`, run continues |

### 1.5 Determinism

- Worker pool results collected into a slice indexed by resource position, so order is independent of completion order.
- `Aggregator.Build` calls `Canonicalize`.
- `Now` injectable; tests pin `GeneratedAt`.

---

## 2. NFR Requirements

| NFR | Obligation |
|---|---|
| NFR-1.1 | resource fan-out bounded pool; 200 resources within budget (client pool + cache do the heavy lifting) |
| NFR-2.1 | all sums via `pricing` `big.Rat` helpers; no float |
| NFR-2.3 (PBT) | PBT-03: aggregation invariants — resource total == Σ components; project total == Σ resource totals; totals ≥ 0; summary counts consistent. PBT-02: usage-file YAML round trip (already in U0; U3 adds a plan-driven skeleton round trip) |
| NFR-3.2 | `PricedResource`/`Breakdown` carry only cost-relevant identifiers; usage values are numbers |
| NFR-5.3 | `internal/engine` imports `plan` + `pricing` (for sum helpers) + `schema`; `internal/usage` imports `plan` + `schema` only |
| NFR-7 | metadata carries query/cache/dedup stats |

### 2.1 Dependencies

`golang.org/x/sync/errgroup` (already have `x/sync`) for the bounded worker pool. No other new deps.

---

## 3. NFR Design

- **Worker pool**: `errgroup.Group` + `SetLimit(n)`; each goroutine writes into `results[i]` (pre-sized), no shared mutable slice append.
- **Querier stats**: accessed through a local `interface{ Stats() pricing.QueryStats }` type assertion so `estimator` doesn't hard-depend on `*pricing.CachingClient`.
- **Prior-state pricing**: a tiny helper `withAttributes(res, attrs)` returns a copy — keeps pricers oblivious.
- **`usage.Model` is an interface** with an `emptyModel` implementation for the `path == ""` case, so callers never nil-check.

### 3.1 File plan

```
internal/usage/doc.go
internal/usage/model.go        Load, Model, emptyModel, Validate, Warnings
internal/usage/keys.go         knownUsageKeys, usageKeysByType (+ descriptions)
internal/usage/skeleton.go     GenerateSkeleton
internal/engine/aggregator.go  PricedResource, Aggregator.Build
internal/estimator/estimator.go Deps, EstimateRequest, Estimate, EstimateBothStates
internal/estimator/pool.go     bounded fan-out helper
*_test.go
```

### 3.2 Test targets (DoD)

- `usage`: load valid file; malformed YAML → error; unknown key → warning; unknown address → warning (via `Validate`); empty path → empty model; `GenerateSkeleton` lists usage keys for in-plan types only and round-trips through `Load`.
- `engine`: aggregate a fixed `[]PricedResource` → expected `Breakdown` (golden compare against `schematest`-style value); module rollups; coverage summary; empty input; **PBT-03** invariants (`rapid`).
- `estimator`: end-to-end with fake `Loader`/`Parser` (or real parser + fixture) + fake `Catalog`/`Querier`:
  - unsupported type → `UNSUPPORTED_TYPE`
  - unresolved region → `MISSING_ATTRIBUTE`
  - happy path → components + totals
  - `EstimateBothStates` → prior vs planned differ; second pass hits warm cache (0 new queries via stubbed stats)
  - metadata: `GeneratedAt` pinned, stats copied
  - determinism: two runs identical bytes
- import-lint for `internal/usage` (no SDK, no `pricing`).

---

## 4. Open items → later units

- Renderers (U4) consume `*schema.Breakdown`.
- `--usage-file`, `--concurrency`, `--period` flags wired in U5; Estimator exposes them on `EstimateRequest`.
- `usage generate` subcommand (U5) calls `usage.GenerateSkeleton`.
