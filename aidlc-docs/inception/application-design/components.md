# Components — Price Checker

High-level component identification. Interfaces are Go sketches; **method-level detail is in `component-methods.md`**, and business rules are deferred to per-unit Functional Design.

## Design decisions carried in (from `application-design-plan.md` + clarification)

| # | Decision |
|---|---|
| Q1 | Hybrid layout: `cmd/price-checker/` main; `internal/` for app packages; exported **`pkg/schema`** holding only the public JSON-output and usage-file types |
| Q2 | CLI framework: **`urfave/cli/v2`** |
| Q3 | Configuration: **hand-rolled resolver** (`internal/config`), explicit `flag > env > file > default`, with provenance tracking |
| Q4 | Money type: **`math/big.Rat`** internally; format to cents (half-up) only at render time |
| Q5 | Pricer catalog: **explicit `NewCatalog() []Pricer`** constructor — no `init()` registration |
| Q6→A | Pricing = **two layers**: shared `PriceListClient` (SDK, credentials, region routing, retry/backoff, bounded worker pool, per-run dedup) + on-disk **cache decorator**; `Pricer`s are thin and never import the AWS SDK |
| Q7 | Rendering: **`Renderer`** + **`DiffRenderer`** interfaces, one implementation per format |
| Q8 | Orchestration: **`Estimator`** and **`Differ`** services; the CLI layer owns input wiring and exit-code/threshold policy |
| Q9 | HTML report: **`html/template`** + CSS/JS inlined via `//go:embed`; one small (~10 KB) charting helper, no external requests |
| Q10 | Public JSON output: **authored JSON Schema** file, `//go:embed`-ed, output validated against it in tests; Go structs in `pkg/schema` are the marshalling source |

## Package map

```
cmd/price-checker/            main(): build CLI app, call cli.Run, os.Exit(code)
pkg/schema/                   PUBLIC types + embedded JSON Schema (Breakdown, DiffResult, UsageFile, ReasonCode, SchemaVersion)
internal/plan/          U1    plan JSON loading + parsing + resource model
internal/pricing/       U2    PriceListClient, cache decorator, Pricer interface, Catalog, pricers, RegionResolver
internal/pricing/aws/   U2    per-resource-type Pricer implementations
internal/usage/         U3    usage-file parse/validate/lookup + skeleton generator
internal/engine/        U3    Aggregator: priced components -> schema.Breakdown, decimal rollups, coverage summary
internal/estimator/     U3/U6 Estimator service (orchestration)
internal/diff/          U5    Differ service + DiffResult construction
internal/render/        U4    Renderer / DiffRenderer interfaces + table/json/html/gh-comment impls
internal/render/html/   U4    embedded template + assets
internal/config/        U6    Config struct + layered resolver + YAML file load + provenance
internal/cli/           U6    urfave/cli command tree, InputResolver, ExitPolicy, logging setup
internal/awsauth/       U2    SDK config / credential chain / Price List endpoint selection
```

---

## U1 — `internal/plan`

### C1. PlanLoader
- **Purpose**: Turn a `--path` value (or stdin) into raw plan JSON bytes.
- **Responsibilities**: detect file vs directory vs `-` (stdin); for a directory, invoke `terraform show -json` (generating a plan file as needed) and capture stdout; clean up temp files; surface a clear error when `terraform` is absent in directory mode.
- **Interface**:
  ```go
  type PlanLoader interface {
      Load(ctx context.Context, source string, stdin io.Reader) ([]byte, LoadInfo, error)
  }
  type LoadInfo struct { Mode string /* "file"|"stdin"|"dir" */; TerraformVersion string }
  ```

### C2. PlanParser
- **Purpose**: Parse plan JSON into the internal resource model.
- **Responsibilities**: validate `format_version` (accept Terraform 1.x); walk `planned_values.root_module` recursively and correlate with `resource_changes`; resolve each resource's attribute map (post-plan values), marking values that are `known after apply`; capture module path and provider (incl. aliased) per resource; classify change action (create/update/delete/no-op/replace).
- **Interface**:
  ```go
  type PlanParser interface {
      Parse(raw []byte) (*Plan, error)
  }
  ```

### C3. Plan model (data types)
- **Purpose**: Provider-neutral representation the rest of the app consumes; nothing downstream imports Terraform JSON shapes.
- **Key types**: `Plan`, `Resource` (`Address`, `Type`, `Name`, `ModulePath`, `ProviderName`, `ProviderAlias`, `Region string`, `ChangeAction`, `Attributes AttrMap`, `PriorAttributes AttrMap`), `AttrMap` with typed getters (`String`, `Int`, `Bool`, `List`, `Map`, `IsKnown(path)`).
- **Note**: `count`/`for_each` are already expanded by Terraform in plan JSON; addresses carry the index key.

---

## U2 — `internal/pricing`, `internal/pricing/aws`, `internal/awsauth`

### C4. AWSConfigProvider (`internal/awsauth`)
- **Purpose**: Resolve AWS SDK configuration once.
- **Responsibilities**: load SDK config via the default credential chain (env, shared config/credentials, profiles, SSO, container/instance roles); pick a Price List API endpoint region (`us-east-1`, fallback `ap-south-1`); produce an actionable error when credentials are missing/invalid.
- **Interface**:
  ```go
  type AWSConfigProvider interface {
      Load(ctx context.Context, profile string) (aws.Config, error)
  }
  ```

### C5. PriceListClient (shared, layer 1)
- **Purpose**: The only component that calls the AWS Price List API.
- **Responsibilities**: execute a `GetProducts` query for a service code + filter set; parse returned product JSON into `[]PriceDimension`; own retry/backoff on throttling; run queries through a **bounded worker pool** (default 8); **deduplicate** identical in-flight/observed queries within a run (singleflight + per-run memo); count queries issued.
- **Interface**:
  ```go
  type PriceQuerier interface {
      Query(ctx context.Context, q PriceQuery) ([]PriceDimension, error)
  }
  type PriceListClient interface {
      PriceQuerier
      Stats() QueryStats // queries issued, dedup hits
  }
  type PriceQuery struct { ServiceCode string; Region string; Filters []Filter }
  type PriceDimension struct { Unit string; PricePerUnit string /* decimal string */; Currency string; Description string; Attributes map[string]string }
  ```

### C6. CachingClient (cache decorator, layer 1)
- **Purpose**: Wrap a `PriceListClient` with the on-disk cache.
- **Responsibilities**: key each `PriceQuery` deterministically; read/write cache entries with timestamp + TTL; honour `--no-cache` (bypass both ways) and `--refresh-cache` (ignore on read, write fresh); atomic writes (temp + rename); 0600/0700 perms; treat corrupt/unreadable entries as misses; never persist plan-derived data (only query keys + Price List responses); expose cache hit/miss stats.
- **Interface**: implements `PriceListClient`; constructed as `NewCachingClient(inner PriceListClient, store CacheStore, opts CacheOptions)`.

### C7. CacheStore
- **Purpose**: Low-level content-addressed disk store.
- **Interface**:
  ```go
  type CacheStore interface {
      Get(key string) (CacheEntry, bool, error)
      Put(key string, entry CacheEntry) error
  }
  type CacheEntry struct { StoredAt time.Time; Payload []byte }
  ```

### C8. Pricer (thin plugin interface, layer 2)
- **Purpose**: Encode how one AWS resource type is priced.
- **Responsibilities**: given a `plan.Resource` and its optional `ResourceUsage`, build the Price List queries it needs, call the injected `PriceQuerier`, select the right dimensions, and emit `CostComponent`s plus `NotEstimated` entries for anything it cannot price. **Never** imports the AWS SDK or the cache.
- **Interface**:
  ```go
  type Pricer interface {
      ResourceTypes() []string
      Price(ctx context.Context, in PricingInput) (PricingOutput, error)
  }
  type PricingInput struct { Resource plan.Resource; Usage schema.ResourceUsage; Q PriceQuerier }
  type PricingOutput struct { Components []schema.CostComponent; NotEstimated []schema.NotEstimated }
  ```

### C9. Catalog
- **Purpose**: The v1 set of pricers and lookup by resource type.
- **Responsibilities**: `NewCatalog()` returns the explicit list (no `init()`); `For(resourceType)` returns the matching pricer or nil (nil → engine records `UNSUPPORTED_TYPE`).
- **Interface**:
  ```go
  type Catalog interface {
      For(resourceType string) (Pricer, bool)
      All() []Pricer
  }
  ```

### C10. RegionResolver
- **Purpose**: Decide which AWS region a resource is priced in.
- **Responsibilities**: read the resource's provider config (incl. alias) from the plan; apply a global `--aws-region` override; return `("", false)` when undeterminable so the engine records `MISSING_ATTRIBUTE`.
- **Interface**: `Resolve(r plan.Resource, override string) (string, bool)`.

### C11. AWS Pricers (`internal/pricing/aws`)
- **Purpose**: One implementation of `Pricer` per catalogued resource group.
- **Members**: `instancePricer`, `ebsVolumePricer`, `ebsSnapshotPricer`, `eipPricer`, `rdsInstancePricer`, `auroraPricer`, `s3Pricer`, `lambdaPricer`, `lbPricer`, `natGatewayPricer`, `eksClusterPricer`, `eksNodeGroupPricer`, `elastiCachePricer`, `dynamoDBPricer`, `cloudWatchPricer`, `dataTransferPricer`.
- **Responsibilities**: pure mapping logic + Price List filter construction; each ships with recorded API fixtures for its unit tests.

---

## U3 — `internal/usage`, `internal/engine`, `internal/estimator`

### C12. UsageModel (`internal/usage`)
- **Purpose**: Load and expose user-supplied monthly usage assumptions.
- **Responsibilities**: parse the YAML usage file (Infracost-style: `version`, `resource_usage` keyed by address); validate keys/addresses and emit **warnings** (not errors) for unknowns; provide `For(address)` lookup returning a `schema.ResourceUsage`.
- **Interface**:
  ```go
  type UsageModel interface {
      For(address string) schema.ResourceUsage
      Warnings() []string
  }
  ```

### C13. UsageSkeletonGenerator (`internal/usage`)
- **Purpose**: Implement `price-checker usage generate`.
- **Responsibilities**: from a parsed `Plan` + `Catalog`, list every usage-based component each resource exposes and emit a commented YAML skeleton with empty values.
- **Interface**: `Generate(p *plan.Plan, cat pricing.Catalog, w io.Writer) error`.

### C14. Aggregator (`internal/engine`)
- **Purpose**: Fold priced components into the final `schema.Breakdown`.
- **Responsibilities**: sum component costs per resource, per module, and project-wide using `big.Rat`; compute hourly and monthly (730 h); collect all `NotEstimated` entries with reason codes; compute the coverage summary (estimated vs not-estimated counts); deterministic ordering; attach run metadata (tool version, timestamp, regions, usage-file used, cache stats).
- **Interface**:
  ```go
  type Aggregator interface {
      Build(priced []PricedResource, meta schema.RunMetadata) *schema.Breakdown
  }
  type PricedResource struct { Resource plan.Resource; Components []schema.CostComponent; NotEstimated []schema.NotEstimated }
  ```

### C15. Estimator (`internal/estimator`) — service
- **Purpose**: End-to-end "plan → Breakdown".
- **Responsibilities**: orchestrate C1→C2→(C10 region)→(C9/C8 pricing via the shared client + worker pool)→(C12 usage)→C14; produce one `schema.Breakdown` (or two, prior + planned, for single-plan diff derivation). See `services.md`.
- **Interface**:
  ```go
  type Estimator interface {
      Estimate(ctx context.Context, req EstimateRequest) (*schema.Breakdown, error)
      EstimateBothStates(ctx context.Context, req EstimateRequest) (prior, planned *schema.Breakdown, err error)
  }
  ```

---

## U4 — `internal/render`

### C16. Renderer / DiffRenderer interfaces
- **Purpose**: Format a `Breakdown` / `DiffResult` to a writer.
- **Interface**:
  ```go
  type Renderer interface { Render(w io.Writer, b *schema.Breakdown, opts Options) error }
  type DiffRenderer interface { RenderDiff(w io.Writer, d *schema.DiffResult, opts Options) error }
  type Options struct { Period Period; ShowComponents bool; Color bool }
  ```

### C17. TableRenderer
- Human-readable table + footer totals + "Not estimated" section; `NO_COLOR`/non-TTY aware; ≤100 cols; optional indented component list.

### C18. JSONRenderer
- Emits `schema.Breakdown` / `schema.DiffResult` as deterministic JSON (stable key + array order); embeds `schema_version`; the output is what the embedded JSON Schema validates in tests.

### C19. HTMLRenderer
- `html/template` with `//go:embed` template + CSS + one small JS helper; sections: summary, per-module, per-resource components, not-estimated, and (diff mode) the change view; embeds run metadata; zero external requests.

### C20. GitHubCommentRenderer
- Markdown: headline total or delta, compact table sorted by |delta|, `<details>` with full breakdown, not-estimated count, tool version/timestamp; deterministic for in-place comment updates.

---

## U5 — `internal/diff`

### C21. Differ — service
- **Purpose**: Compute the cost change between two `Breakdown`s.
- **Responsibilities**: match resources by address across base/proposed; classify added / removed / changed / unchanged; compute per-resource and total deltas (`big.Rat`); carry not-estimated status through rather than inventing deltas; build `schema.DiffResult`.
- **Interface**:
  ```go
  type Differ interface { Diff(base, proposed *schema.Breakdown) *schema.DiffResult }
  ```

---

## U6 — `internal/config`, `internal/cli`, `cmd/price-checker`

### C22. Config + Resolver (`internal/config`)
- **Purpose**: One resolved settings object with known provenance.
- **Responsibilities**: define `Config` (all global options); load optional `price-checker.yml`; layer `flag > env (PRICE_CHECKER_*) > file > default`; record where each value came from; warn (not fail) on unknown file keys; expose the effective config for `--log-level debug`.
- **Interface**:
  ```go
  type Resolver interface {
      Resolve(flags FlagSet, env Environ, filePath string) (Config, Provenance, error)
  }
  ```

### C23. InputResolver (`internal/cli`)
- **Purpose**: Interpret `--path` / `--compare-to` values.
- **Responsibilities**: classify each as plan-JSON file, Terraform directory, prior price-checker JSON output, or stdin; hand the right loader to the `Estimator`/`Differ`.

### C24. CLI command tree (`internal/cli`)
- **Purpose**: `urfave/cli/v2` app: `breakdown` (default), `diff`, `report`, `usage generate`, `version`, `help`.
- **Responsibilities**: define flags (FR-12.2), bind them into `config.Resolver`, construct services, select `Renderer`/`DiffRenderer` by `--format`, route output to stdout or `--out`, keep logs on stderr.

### C25. ExitPolicy (`internal/cli`)
- **Purpose**: Map an outcome to a process exit code.
- **Responsibilities**: `0` success; `1` runtime error; `2` `--strict` with any not-estimated component; `3` threshold breach (`--threshold-monthly`, `--threshold-diff-monthly`); precedence `1 > 2 > 3`; always render full output before exiting non-zero for 2/3.
- **Interface**: `Evaluate(result Outcome, cfg Config) int`.

### C26. Logging (`internal/cli`)
- **Purpose**: Level-controlled diagnostics on stderr only (`log/slog`), never polluting stdout.

### C27. main (`cmd/price-checker`)
- Thin: build the CLI app, run it, `os.Exit` with the policy's code.

---

## `pkg/schema` (public, exported)

- **Types**: `Breakdown`, `Resource`, `CostComponent`, `ModuleTotal`, `NotEstimated`, `ReasonCode` (`UNSUPPORTED_TYPE`, `MISSING_ATTRIBUTE`, `UNKNOWN_AFTER_APPLY`, `NO_USAGE_DATA`, `PRICING_API_ERROR`, `UNSUPPORTED_CONFIGURATION`), `CoverageSummary`, `RunMetadata`, `DiffResult`, `ResourceDelta`, `UsageFile`, `ResourceUsage`.
- **Constants**: `SchemaVersion`, `MonthlyHours = 730`.
- **Embedded**: `breakdown.schema.json`, `diff.schema.json`, `usage.schema.json` (`//go:embed`), plus a `Validate` helper used by tests.
- **Rule**: this is the only package `internal/*` may expose outward; money is serialized as a decimal **string**, never a float.

---

## Cross-cutting ownership

| Concern | Owning component(s) |
|---|---|
| No plan data leaves host / cache scrubbing (NFR-3) | C6 CachingClient (never persists plan data), C14 Aggregator (Breakdown carries only cost-relevant identifiers) |
| Determinism (NFR-2.1) | C14 Aggregator (ordering), C18 JSONRenderer (stable encoding), C21 Differ |
| Concurrency + dedup (NFR-1.2) | C5 PriceListClient (worker pool + singleflight) |
| Property-based tests (NFR-2.3) | `pkg/schema` round-trips, C14 aggregation invariants, C8 dimension parsing |
| Extensibility (NFR-5.1) | C8 Pricer + C9 Catalog; C16 renderers depend only on `pkg/schema` |
| Portability (NFR-6) | C27 main + build config; no cgo; `//go:embed` assets |
