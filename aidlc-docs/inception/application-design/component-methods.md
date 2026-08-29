# Component Methods — Price Checker

Method signatures, input/output types, and one-line purpose. **Business rules (rounding modes, exact filter sets, per-resource cost formulas, validation specifics) are defined in per-unit Functional Design, not here.**

Conventions: all I/O-bound methods take `context.Context`. Money is `*big.Rat` internally and a decimal `string` in `pkg/schema`. Errors follow Go conventions; "cannot price this" is **not** an error — it is a `schema.NotEstimated` entry.

---

## U1 — `internal/plan`

### PlanLoader
| Method | Signature | Purpose |
|---|---|---|
| Load | `Load(ctx, source string, stdin io.Reader) ([]byte, LoadInfo, error)` | Return raw plan JSON from a file, stdin (`-`/empty), or a Terraform directory (runs `terraform show -json`). |

### PlanParser
| Method | Signature | Purpose |
|---|---|---|
| Parse | `Parse(raw []byte) (*Plan, error)` | Validate `format_version` and build the `*Plan` model from `planned_values` + `resource_changes`. |

### Plan model
| Method | Signature | Purpose |
|---|---|---|
| Plan.Resources | `Resources() []Resource` | All planned (non-deleted for cost) resources, deterministically ordered by address. |
| Plan.PriorResources | `PriorResources() []Resource` | Prior-state resources (for single-plan diff derivation). |
| Resource.Attr | `Attr() AttrMap` | Resolved post-plan attribute map. |
| AttrMap.String / Int / Bool | `String(path string) (string, bool)` … | Typed attribute access; second return is `known`. |
| AttrMap.IsKnown | `IsKnown(path string) bool` | False when the value is "known after apply". |
| AttrMap.List / Map | `List(path string) ([]AttrMap, bool)` / `Map(path string) (AttrMap, bool)` | Nested-structure access. |

---

## U2 — `internal/awsauth`, `internal/pricing`, `internal/pricing/aws`

### AWSConfigProvider
| Method | Signature | Purpose |
|---|---|---|
| Load | `Load(ctx, profile string) (aws.Config, error)` | Resolve SDK config via the default credential chain; error message names chain options + `--aws-region` on failure. |
| PriceListEndpointRegion | `PriceListEndpointRegion(cfg aws.Config) string` | Pick `us-east-1` (or `ap-south-1` fallback) for Price List API calls. |

### PriceListClient / PriceQuerier
| Method | Signature | Purpose |
|---|---|---|
| Query | `Query(ctx, q PriceQuery) ([]PriceDimension, error)` | Run one `GetProducts` call (through the worker pool + dedup) and parse products into price dimensions. |
| Stats | `Stats() QueryStats` | Queries issued, dedup hits, throttle-retries for the run. |
| Close | `Close() error` | Drain and stop the worker pool. |

### CachingClient (decorator; also implements PriceListClient)
| Method | Signature | Purpose |
|---|---|---|
| NewCachingClient | `NewCachingClient(inner PriceListClient, store CacheStore, opts CacheOptions) *CachingClient` | Wrap the shared client with TTL disk caching. |
| Query | `Query(ctx, q PriceQuery) ([]PriceDimension, error)` | Serve from cache when fresh & allowed; otherwise delegate and store. |
| Stats | `Stats() QueryStats` | Adds cache hit/miss ratio to the inner stats. |

### CacheStore
| Method | Signature | Purpose |
|---|---|---|
| Get | `Get(key string) (CacheEntry, bool, error)` | Load an entry; `false` on miss; corrupt entry treated as miss (no error). |
| Put | `Put(key string, entry CacheEntry) error` | Atomic write (temp + rename), 0600 perms. |
| KeyFor | `KeyFor(q PriceQuery) string` | Deterministic content-addressed key for a query. |

### Pricer (plugin interface)
| Method | Signature | Purpose |
|---|---|---|
| ResourceTypes | `ResourceTypes() []string` | Terraform resource type strings this pricer handles. |
| Price | `Price(ctx, in PricingInput) (PricingOutput, error)` | Build queries, call `in.Q`, select dimensions, emit `CostComponent`s + `NotEstimated`s. Error only for programmer/contract faults. |

### Catalog
| Method | Signature | Purpose |
|---|---|---|
| NewCatalog | `NewCatalog() Catalog` | Construct the explicit v1 pricer list (no `init()`). |
| For | `For(resourceType string) (Pricer, bool)` | Lookup; miss → engine records `UNSUPPORTED_TYPE`. |
| All | `All() []Pricer` | Every registered pricer (used by the usage-skeleton generator). |

### RegionResolver
| Method | Signature | Purpose |
|---|---|---|
| Resolve | `Resolve(r plan.Resource, override string) (string, bool)` | Resource region from provider config; `override` wins; `false` → `MISSING_ATTRIBUTE`. |

### AWS pricers (each type, e.g. instancePricer)
| Method | Signature | Purpose |
|---|---|---|
| ResourceTypes | `ResourceTypes() []string` | e.g. `["aws_instance"]`. |
| Price | `Price(ctx, in PricingInput) (PricingOutput, error)` | Per-resource pricing logic (formula detail → Functional Design). |

---

## U3 — `internal/usage`, `internal/engine`, `internal/estimator`

### UsageModel
| Method | Signature | Purpose |
|---|---|---|
| Load | `Load(path string) (UsageModel, error)` | Parse + validate the usage YAML; malformed YAML is a hard error. |
| For | `For(address string) schema.ResourceUsage` | Usage assumptions for a resource address (zero value when absent). |
| Warnings | `Warnings() []string` | Non-fatal messages (unknown address/key). |

### UsageSkeletonGenerator
| Method | Signature | Purpose |
|---|---|---|
| Generate | `Generate(p *plan.Plan, cat pricing.Catalog, w io.Writer) error` | Emit a commented YAML skeleton listing usage-based components per resource. |

### Aggregator
| Method | Signature | Purpose |
|---|---|---|
| Build | `Build(priced []PricedResource, meta schema.RunMetadata) *schema.Breakdown` | Roll component costs up to resource / module / project totals (hourly + monthly), collect not-estimated, compute coverage summary, attach metadata, deterministic order. |

### Estimator (service)
| Method | Signature | Purpose |
|---|---|---|
| Estimate | `Estimate(ctx, req EstimateRequest) (*schema.Breakdown, error)` | Full pipeline: load → parse → resolve regions → price (pooled) → merge usage → aggregate. |
| EstimateBothStates | `EstimateBothStates(ctx, req EstimateRequest) (prior, planned *schema.Breakdown, error)` | Same, producing prior + planned breakdowns from one plan for `diff --from-plan`. |

`EstimateRequest`: `{ Source string; Stdin io.Reader; RegionOverride string; UsagePath string; CacheOptions; Concurrency int; Strict bool }`.

---

## U4 — `internal/render`

### Renderer implementations (TableRenderer, JSONRenderer, HTMLRenderer, GitHubCommentRenderer)
| Method | Signature | Purpose |
|---|---|---|
| Render | `Render(w io.Writer, b *schema.Breakdown, opts Options) error` | Write the breakdown in that format. |

### DiffRenderer implementations (Table, JSON, HTML, GitHubComment)
| Method | Signature | Purpose |
|---|---|---|
| RenderDiff | `RenderDiff(w io.Writer, d *schema.DiffResult, opts Options) error` | Write the diff in that format. |

### Factory
| Method | Signature | Purpose |
|---|---|---|
| RendererFor | `RendererFor(format string) (Renderer, DiffRenderer, error)` | Resolve `--format` to the pair of renderers (some formats implement both). |

---

## U5 — `internal/diff`

### Differ (service)
| Method | Signature | Purpose |
|---|---|---|
| Diff | `Diff(base, proposed *schema.Breakdown) *schema.DiffResult` | Match by address; classify added/removed/changed/unchanged; compute per-resource + total deltas; preserve not-estimated status. |

---

## U6 — `internal/config`, `internal/cli`, `cmd/price-checker`

### config.Resolver
| Method | Signature | Purpose |
|---|---|---|
| Resolve | `Resolve(flags FlagSet, env Environ, filePath string) (Config, Provenance, error)` | Layer flag > env > file > default; warn on unknown file keys. |
| Provenance.Source | `Source(key string) string` | Where a given setting's value came from (for `--log-level debug`). |

### cli.InputResolver
| Method | Signature | Purpose |
|---|---|---|
| Classify | `Classify(value string) (InputKind, error)` | `planFile` / `terraformDir` / `priorJSON` / `stdin`. |

### cli.App
| Method | Signature | Purpose |
|---|---|---|
| NewApp | `NewApp(deps Deps) *cli.App` | Build the `urfave/cli` command tree with all flags bound. |
| Run | `Run(ctx, args []string, stdin io.Reader, stdout, stderr io.Writer) int` | Execute; return the process exit code. |

### cli.ExitPolicy
| Method | Signature | Purpose |
|---|---|---|
| Evaluate | `Evaluate(o Outcome, cfg Config) int` | Outcome + flags → `0/1/2/3` with precedence `1 > 2 > 3`. |

`Outcome`: `{ Err error; Breakdown *schema.Breakdown; Diff *schema.DiffResult; NotEstimatedCount int; MonthlyTotal *big.Rat; MonthlyDelta *big.Rat }`.

### cli.Logging
| Method | Signature | Purpose |
|---|---|---|
| Setup | `Setup(level string, stderr io.Writer) *slog.Logger` | Configure stderr-only levelled logging. |

### main
| Method | Signature | Purpose |
|---|---|---|
| main | `func main()` | `os.Exit(cli.NewApp(deps).Run(...))`. |

---

## `pkg/schema`

| Method / value | Signature | Purpose |
|---|---|---|
| SchemaVersion | `const SchemaVersion = "1.0"` | Bumped on any breaking output change. |
| MonthlyHours | `const MonthlyHours = 730` | Hour→month conversion constant. |
| Breakdown.MarshalJSON | `MarshalJSON() ([]byte, error)` | Deterministic encoding (stable key/array order), money as decimal strings. |
| UsageFile round-trip | `ParseUsageFile([]byte) (UsageFile, error)` / `(UsageFile).Marshal() ([]byte, error)` | Lossless YAML round-trip (PBT-02 target). |
| Validate | `Validate(kind string, doc []byte) error` | Validate a document against the embedded JSON Schema (test + optional `--validate`). |
