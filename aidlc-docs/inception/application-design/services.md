# Services — Price Checker

Per design decision Q8 = A, orchestration lives in **two service types** — `Estimator` and `Differ` — and the **CLI layer** owns input wiring and exit-code/threshold **policy**. Services are pure orchestration: they take resolved inputs and return domain results; they do not read flags, touch `os.Exit`, or write to stdout.

---

## Service 1 — `Estimator` (`internal/estimator`)

### Responsibility
Turn a plan source into one or two `schema.Breakdown` values. Owns the pricing pipeline and the shared worker pool for a run.

### Dependencies (injected)
- `plan.PlanLoader`, `plan.PlanParser`
- `pricing.RegionResolver`
- `pricing.Catalog`
- `pricing.PriceListClient` (already wrapped with the cache decorator by the CLI layer)
- `usage.UsageModel` (may be the empty model)
- `engine.Aggregator`
- `*slog.Logger`

### Orchestration sequence — `Estimate`

```
1. raw, info  := PlanLoader.Load(ctx, req.Source, req.Stdin)
2. plan       := PlanParser.Parse(raw)                     // err -> exit 1 upstream
3. resources  := plan.Resources()
4. for each resource (bounded worker pool, size = req.Concurrency):
     a. region, ok := RegionResolver.Resolve(res, req.RegionOverride)
        if !ok -> record NotEstimated{MISSING_ATTRIBUTE}; continue
     b. pricer, ok := Catalog.For(res.Type)
        if !ok -> record NotEstimated{UNSUPPORTED_TYPE}; continue
     c. usage := UsageModel.For(res.Address)
     d. out, err := pricer.Price(ctx, PricingInput{Resource: res, Usage: usage, Q: client})
        if err (contract fault) -> record NotEstimated{PRICING_API_ERROR or UNSUPPORTED_CONFIGURATION}
        else -> collect out.Components + out.NotEstimated
   (Price List calls inside pricer.Price go through the shared client:
    dedup + retry/backoff + throttle handling live there, not here.)
5. meta := RunMetadata{tool version, timestamp, regions seen, usage-file used, client.Stats()}
6. return Aggregator.Build(pricedResources, meta)
```

### `EstimateBothStates`
Runs the resource loop twice — once over `plan.PriorResources()`, once over `plan.Resources()` — reusing the same client/cache so the second pass is almost all cache hits. Returns `(prior, planned)` for `diff --from-plan`.

### Error vs not-estimated
| Situation | Handling |
|---|---|
| Plan unreadable / bad format / malformed JSON | return `error` → CLI exit 1 |
| Credentials missing/invalid | return `error` → CLI exit 1 |
| Single resource unsupported / missing attr / unknown-after-apply / no usage data | `schema.NotEstimated` entry, run continues |
| Price List query fails after retries | `schema.NotEstimated{PRICING_API_ERROR}` for affected resources, run continues |
| Context cancelled / deadline | return `error` → CLI exit 1 |

---

## Service 2 — `Differ` (`internal/diff`)

### Responsibility
Compute the cost change between two `schema.Breakdown` values. Pure function, no I/O.

### Dependencies
None (operates on in-memory breakdowns). Uses `math/big` for delta arithmetic.

### Orchestration sequence — `Diff(base, proposed)`

```
1. index base and proposed resources by address
2. for each address in union(base, proposed):
     - in proposed only        -> ResourceDelta{kind: added,   prior: 0, new: cost}
     - in base only            -> ResourceDelta{kind: removed, prior: cost, new: 0}
     - in both, cost differs   -> ResourceDelta{kind: changed}
     - in both, cost equal     -> ResourceDelta{kind: unchanged}
     - not-estimated on either side -> mark delta not-estimated, do not fabricate a number
3. totalDelta = sum(new) - sum(prior) over estimated resources (monthly + hourly)
4. return schema.DiffResult{Changes, TotalDelta*, Coverage, Metadata}
```

---

## Policy boundary — owned by the CLI layer (`internal/cli`), NOT the services

| Policy | Component | Notes |
|---|---|---|
| Which input is a file / dir / prior-JSON / stdin | `InputResolver` | Feeds the right source string / reader to the service |
| Wrapping `PriceListClient` with the cache decorator + options | `cli.App` wiring | Services receive the already-wrapped client |
| `--format` → concrete `Renderer` / `DiffRenderer` | `render.RendererFor` via `cli.App` | |
| stdout vs `--out` file; logs to stderr | `cli.App` + `cli.Logging` | Services never write output |
| Exit code `0/1/2/3`, `--strict`, thresholds | `cli.ExitPolicy` | Runs after the service returns |
| Config precedence (`flag > env > file > default`) | `config.Resolver` | Produces the `Config` the wiring reads |

---

## End-to-end flow per subcommand

### `price-checker breakdown` (default)
`config.Resolve` → `InputResolver.Classify(--path)` → build client (+cache) → `Estimator.Estimate` → `RendererFor(--format).Render` → `stdout`/`--out` → `ExitPolicy.Evaluate` (strict / threshold-monthly) → `os.Exit`.

### `price-checker diff`
`config.Resolve` → classify `--path` and `--compare-to` (or `--from-plan`) → build client (+cache):
- prior-JSON inputs are unmarshalled directly to `schema.Breakdown`
- plan/dir inputs go through `Estimator.Estimate`
- `--from-plan` uses `Estimator.EstimateBothStates`
→ `Differ.Diff(base, proposed)` → `RendererFor(--format).RenderDiff` → output → `ExitPolicy.Evaluate` (threshold-diff-monthly) → exit.

### `price-checker report`
Same as `breakdown`/`diff` but forces `--format html` and defaults `--out` to a file.

### `price-checker usage generate`
`config.Resolve` → load + parse plan (no pricing calls) → `UsageSkeletonGenerator.Generate(plan, Catalog)` → output → exit 0.

### `price-checker version`
Print embedded build metadata (version, commit, date, Go version); works offline; exit 0.
