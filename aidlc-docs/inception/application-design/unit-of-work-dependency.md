# Unit of Work Dependency — Price Checker

## Dependency matrix (row depends on column)

| ↓ / → | U0 schema | U1 plan-ingest | U2 pricing-core | U3 cost-engine | U4 output-and-diff | U5 cli-app |
|---|:--:|:--:|:--:|:--:|:--:|:--:|
| **U0 schema** | — |  |  |  |  |  |
| **U1 plan-ingest** | ✅ | — |  |  |  |  |
| **U2 pricing-core** | ✅ | ✅ | — |  |  |  |
| **U3 cost-engine** | ✅ | ✅ | ✅ | — |  |  |
| **U4 output-and-diff** | ✅ |  |  |  | — |  |
| **U5 cli-app** | ✅ | ✅ | ✅ | ✅ | ✅ | — |

Notable: **U4 depends on U0 only** (renderers + diff see just the contract). U5 wires everything.

## Build order

```
U0 schema
   |
   v
U1 plan-ingest ----------------------+
   |                                 |
   v                                 |
U2 pricing-core                      |  (U5 walking skeleton can start here:
   |                                 |   main + `version` + stub `breakdown`)
   v                                 |
U3 cost-engine                       |
   |                                 |
   v                                 v
U4 output-and-diff  <--- depends only on U0, so it can be built any time after U0
   |
   v
U5 cli-app  (full)  <--- needs U0..U4 complete
```

- **Critical path**: U0 → U1 → U2 → U3 → (U4) → U5.
- **U4 float**: because U4 only needs U0, it may be built in parallel with U1–U3 once the `schema.Breakdown` shape is frozen at the end of U0. Recommended: freeze U0, then run U1→U2→U3 while U4 proceeds alongside; converge at U5.
- **U5 walking-skeleton increment**: built immediately after U1 (needs only U0 + U1's `Plan` model); the full U5 completes after U4.

## Parallelization notes

| Can run in parallel | Condition |
|---|---|
| U4 alongside U1/U2/U3 | `pkg/schema` (U0) frozen; U4 test fixtures are hand-authored `schema.Breakdown` values, not real pricing output |
| U5 walking skeleton alongside U2 | Skeleton only imports U0 + U1 |
| Individual pricers within U2 | Each pricer + its fixtures is independent once `Pricer`/`PriceQuerier`/`Catalog` exist |
| Individual renderers within U4 | Each renderer is independent once the `Renderer`/`DiffRenderer` interfaces + a shared sample breakdown exist |

The AI-DLC per-unit loop still processes one unit at a time for design/approval; the parallelism above is about implementation effort, not the workflow's approval gates.

## Integration points (contracts between units)

| Boundary | Contract | Owned by |
|---|---|---|
| U1 → U2/U3 | `plan.Plan`, `plan.Resource`, `plan.AttrMap` (typed getters, `IsKnown`) | U1 |
| U0 → all | `pkg/schema` types + `SchemaVersion` + embedded JSON Schemas | U0 |
| U2 → U3 | `pricing.Catalog`, `pricing.Pricer`, `pricing.PricingOutput` (`[]schema.CostComponent`, `[]schema.NotEstimated`), `pricing.RegionResolver`, `pricing.PriceListClient` | U2 |
| U2 internal | `PriceQuerier` (pricers ↔ shared client); `CacheStore` (decorator ↔ disk) | U2 |
| U3 → U4/U5 | `*schema.Breakdown` (from `Aggregator`/`Estimator`) | U3 (shape from U0) |
| U4 → U5 | `render.Renderer`, `render.DiffRenderer`, `render.RendererFor`, `diff.Differ` | U4 |
| U5 → process | exit codes `0/1/2/3`, stdout = data, stderr = logs | U5 |
| U5 → CI consumers | JSON schema conformance, stable exit codes, `price-checker.yml` keys | U5 + U0 |

## Risk / coordination

| Risk | Mitigation |
|---|---|
| `schema.Breakdown` shape churns after U4 starts | Freeze and tag U0 before any downstream unit; changes to U0 after freeze require re-review of U3/U4 |
| Price List filter sets wrong (discovered late) | Per-pricer recorded fixtures in U2; a manual "sanity price" check against the AWS calculator during U2 Functional Design |
| Walking skeleton diverges from final U5 | Skeleton lives in the same files it will grow into; no throwaway package |
| U2 is large (16 pricers) | Pricer framework + first pricer reviewed together; remaining pricers are mechanical repeats of the same interface |
