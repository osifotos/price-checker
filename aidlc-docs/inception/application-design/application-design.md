# Application Design — Price Checker (Consolidated)

Consolidates `components.md`, `component-methods.md`, `services.md`, and `component-dependency.md`. Scope: high-level structure and interfaces. Detailed business logic is defined per unit in CONSTRUCTION → Functional Design.

---

## 1. Overview

`price-checker` is a single-binary Go CLI that reads a Terraform plan (JSON file, stdin, or a directory it runs `terraform show -json` against), prices the AWS resources in it against the **live AWS Price List API**, and prints a monthly + hourly USD cost breakdown. It supports a usage file for usage-based components, a `diff` mode, four output formats (table, JSON, HTML, GitHub comment), an on-disk price cache, and CI-friendly exit codes and cost-threshold gates.

**Architecture style**: ports & adapters. A dependency-free core (`pkg/schema` contract types + `internal/engine` aggregation) is surrounded by adapters — Terraform plan parsing (in), AWS Price List pricing (out), and output renderers (out). Two thin services (`Estimator`, `Differ`) orchestrate; the CLI layer owns input classification and exit-code/threshold policy.

---

## 2. Key design decisions and rationale

| # | Decision | Rationale | Trade-off accepted |
|---|---|---|---|
| Q1 | Hybrid layout: `cmd/` + `internal/` + exported `pkg/schema` | Keeps the app private but publishes the JSON-output / usage-file types so CI consumers can depend on them | Slightly more thought about what belongs in `pkg/` |
| Q2 | `urfave/cli/v2` | Lightweight subcommand CLI without pulling in `viper`; matches the "lean dependencies" lean of Q3/Q4 | Less turnkey flag/env binding than cobra+viper — handled by `internal/config` |
| Q3 | Hand-rolled config resolver | Explicit `flag > env > file > default` precedence and **provenance** ("where did this value come from") for `--log-level debug`; zero extra dependency | ~1 file of layering code to own and test |
| Q4 | `math/big.Rat` internally | Exact arithmetic straight from AWS decimal price strings; no float accumulation error (NFR-2.1) | Verbose formatting; a rounding/format policy is defined in `U3` Functional Design (half-up to cents for display, full precision internally) |
| Q5 | Explicit `NewCatalog() []Pricer` | Deterministic catalog, trivial to construct a subset in tests, no `init()` ordering surprises | Adding a pricer means editing one `catalog.go` line (acceptable, and visible in review) |
| Q6→A | Two pricing layers: shared `PriceListClient` (+ cache decorator) and thin `Pricer` | Satisfies dedup + bounded worker pool (NFR-1.2), one on-disk cache with unified stats (FR-4, NFR-7.2), and a one-method extension interface (NFR-5.1); pricers are unit-testable without SDK mocks | More upfront wiring than "each pricer does its own thing" |
| Q7 | `Renderer` + `DiffRenderer` interfaces, one impl per format | Each format isolated; formats that support both breakdown and diff implement both interfaces | Four small files instead of one switch |
| Q8 | `Estimator` + `Differ` services; CLI owns policy | Services stay pure (testable, no `os.Exit`, no flag reads); policy (exit codes, thresholds, stdout vs `--out`) is all in one place | Two layers to trace instead of one |
| Q9 | `html/template` + `//go:embed` CSS/JS, one small chart helper | Self-contained report, zero external requests (FR-10.1) | Binary carries the embedded assets (~tens of KB); specific JS lib vetoable at code-gen planning |
| Q10 | Authored JSON Schema, embedded, validated in tests | The schema file is the consumer contract; tests assert real output conforms; a `schema_version` gates breaking changes | Schema file kept in sync with structs by a test |

---

## 3. Components by unit

| Unit | Package(s) | Components | Delivers stories |
|---|---|---|---|
| **U1 plan-ingest** | `internal/plan` | PlanLoader, PlanParser, Plan model | US-LOCAL-ESTIMATE-1, -2 |
| **U2 pricing-core** | `internal/awsauth`, `internal/pricing`, `internal/pricing/aws` | AWSConfigProvider, PriceListClient, CachingClient, CacheStore, Pricer, Catalog, RegionResolver, 16 AWS pricers | US-PRICING-1…7, US-CACHE-1…3 |
| **U3 cost-engine** | `internal/usage`, `internal/engine`, `internal/estimator` | UsageModel, UsageSkeletonGenerator, Aggregator, Estimator | US-LOCAL-ESTIMATE-3, -4, US-USAGE-1…3, US-PRICING-5 |
| **U4 output-renderers** | `internal/render`, `internal/render/html` | Renderer/DiffRenderer + Table/JSON/HTML/GitHubComment, RendererFor | US-MACHINE-OUTPUT-1, -2, US-REPORT-1, US-PR-COMMENT-1, -2, US-LOCAL-ESTIMATE-3, -4 |
| **U5 diff** | `internal/diff` | Differ | US-DIFF-1, -2, US-PR-COMMENT-2 |
| **U6 cli-app** | `internal/config`, `internal/cli`, `cmd/price-checker` | Config/Resolver, InputResolver, App, ExitPolicy, Logging, main | US-CI-GATE-1, -2, US-CONFIG-INSTALL-1…3, US-PR-COMMENT-3 (later) |
| **shared** | `pkg/schema` | contract types, constants, embedded JSON Schemas | contract for all machine output |

---

## 4. Primary interfaces (sketch)

```go
// pkg/schema — the contract
const (SchemaVersion = "1.0"; MonthlyHours = 730)
type Breakdown struct { SchemaVersion string; Currency string; Resources []Resource;
    Modules []ModuleTotal; TotalMonthly, TotalHourly string; NotEstimated []NotEstimated;
    Summary CoverageSummary; Metadata RunMetadata }
type NotEstimated struct { Address string; ReasonCode ReasonCode; Message string; Component string }

// internal/plan
type PlanLoader interface { Load(ctx, source string, stdin io.Reader) ([]byte, LoadInfo, error) }
type PlanParser interface { Parse(raw []byte) (*Plan, error) }

// internal/pricing
type PriceQuerier interface { Query(ctx, q PriceQuery) ([]PriceDimension, error) }
type PriceListClient interface { PriceQuerier; Stats() QueryStats; Close() error }
type Pricer interface { ResourceTypes() []string; Price(ctx, in PricingInput) (PricingOutput, error) }
type Catalog interface { For(resourceType string) (Pricer, bool); All() []Pricer }
type RegionResolver interface { Resolve(r plan.Resource, override string) (string, bool) }

// internal/engine + internal/estimator
type Aggregator interface { Build(priced []PricedResource, meta schema.RunMetadata) *schema.Breakdown }
type Estimator interface {
    Estimate(ctx, req EstimateRequest) (*schema.Breakdown, error)
    EstimateBothStates(ctx, req EstimateRequest) (prior, planned *schema.Breakdown, error)
}

// internal/diff
type Differ interface { Diff(base, proposed *schema.Breakdown) *schema.DiffResult }

// internal/render
type Renderer interface { Render(w io.Writer, b *schema.Breakdown, opts Options) error }
type DiffRenderer interface { RenderDiff(w io.Writer, d *schema.DiffResult, opts Options) error }

// internal/cli
type ExitPolicy interface { Evaluate(o Outcome, cfg Config) int }
```

---

## 5. Requirements coverage (component-level)

| Requirement | Owning components |
|---|---|
| FR-1 plan ingestion | PlanLoader, PlanParser, Plan model |
| FR-2 pricing catalog | Catalog, AWS pricers, UsageModel |
| FR-3 pricing retrieval | AWSConfigProvider, PriceListClient, RegionResolver |
| FR-4 cache | CachingClient, CacheStore |
| FR-5 usage file | UsageModel, UsageSkeletonGenerator |
| FR-6 cost engine | Aggregator (+ `big.Rat`), Estimator |
| FR-7 table output | TableRenderer |
| FR-8 JSON output | JSONRenderer, `pkg/schema` + embedded schema |
| FR-9 diff | Differ, Estimator.EstimateBothStates, InputResolver |
| FR-10 HTML report | HTMLRenderer, `internal/render/html` |
| FR-11 CI integration | ExitPolicy, GitHubCommentRenderer, config.Resolver |
| FR-12 CLI surface | cli.App, config.Resolver, Logging |
| FR-13 coverage reporting | Aggregator, `schema.NotEstimated` / `ReasonCode` |
| NFR-1 performance | PriceListClient (pool + dedup), CachingClient |
| NFR-2 reliability/correctness | Aggregator (`big.Rat`, ordering), JSONRenderer, Differ |
| NFR-2.3 property-based testing | `pkg/schema` round-trips, Aggregator invariants, PriceDimension parsing |
| NFR-3 security/privacy | CachingClient (no plan data persisted), Aggregator (cost-relevant fields only), AWSConfigProvider |
| NFR-4 usability | AWSConfigProvider error text, cli.App help, TableRenderer |
| NFR-5 maintainability/extensibility | Pricer + Catalog, renderer/`pkg/schema` isolation |
| NFR-6 portability | `cmd/price-checker` + build config, `//go:embed` assets, no cgo |
| NFR-7 observability | Logging, QueryStats in RunMetadata |

**Completeness check**: every FR (1–13) and NFR (1–7) has at least one owning component; every one of the 30 stories maps to a unit in §3; every component belongs to exactly one unit; the dependency graph in `component-dependency.md` is acyclic.

---

## 6. Third-party dependencies (proposed; finalized in per-unit NFR Requirements)

| Purpose | Library | Notes |
|---|---|---|
| CLI | `github.com/urfave/cli/v2` | Q2 |
| AWS Price List | `github.com/aws/aws-sdk-go-v2` + `service/pricing` | only in `internal/awsauth` + `internal/pricing` |
| Query dedup | `golang.org/x/sync/singleflight` | in PriceListClient |
| YAML | `gopkg.in/yaml.v3` | usage file, config file |
| Decimal | `math/big` (stdlib) | Q4 |
| Property-based testing | `pgregory.net/rapid` | PBT-09 |
| JSON Schema validation (tests) | `github.com/santhosh-tekuri/jsonschema/v5` | test-only |
| HTML | `html/template` (stdlib) + embedded assets | Q9 |
| Logging | `log/slog` (stdlib) | stderr only |

---

## 7. Open items deferred to CONSTRUCTION

- Exact Price List `ServiceCode` + filter sets per resource type → per-pricer Functional Design + recorded fixtures.
- Rounding/format policy for `big.Rat` → `U3` Functional Design.
- Usage-file key vocabulary (full list) → `U3` Functional Design, aligned with the Infracost schema.
- Cache key canonicalization and on-disk layout → `U2` NFR Design.
- Worker-pool sizing / backoff parameters and their config flags → `U2` NFR Design.
- The specific embedded JS chart helper → `U4` Code Generation planning (user veto point).
- GitHub Actions recipe (US-PR-COMMENT-3, tagged `later`).
