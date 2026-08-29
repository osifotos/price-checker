# U2 `pricing-core` — Consolidated Design

**Unit**: U2 · **Packages**: `internal/awsauth`, `internal/pricing`, `internal/pricing/aws`
**Depends on**: U0 `pkg/schema`, U1 `internal/plan`
**Owns**: US-PRICING-1..7, US-CACHE-1..3 · **Supports**: US-USAGE-3, US-CI-GATE-2, US-CONFIG-INSTALL-2

---

## 1. Functional Design

### 1.1 Two layers (per Application Design Q6→A)

```
Pricer (per resource type)         <- internal/pricing/aws, thin, no AWS SDK import
    | uses
PriceQuerier                       <- interface
    | implemented by
CachingClient (decorator)          <- internal/pricing, on-disk TTL cache
    | wraps
PriceListClient (shared)           <- internal/pricing, worker pool + dedup + backoff
    | uses
pricing.Client (aws-sdk-go-v2)     <- internal/pricing, GetProducts
```

`internal/awsauth` loads SDK config (credential chain) and picks the Price List endpoint region.

### 1.2 Core types (`internal/pricing`)

```go
type PriceQuery struct {
    ServiceCode string     // "AmazonEC2", "AmazonRDS", ...
    RegionCode  string     // "us-east-1"
    Filters     []Filter   // term-match attribute filters
    Purpose     string     // human label for logs, not part of the cache key
}
type Filter struct { Field, Value string }

type PriceDimension struct {
    SKU          string
    Unit         string   // "Hrs", "GB-Mo", "Requests"
    USD          string   // decimal string, price per unit
    Description  string
    Attributes   map[string]string  // product attributes (instanceType, volumeType, ...)
}

type PriceQuerier interface {
    Query(ctx context.Context, q PriceQuery) ([]PriceDimension, error)
}
```

### 1.3 Pricer interface (`internal/pricing`)

```go
type PricingInput struct {
    Resource plan.Resource
    Usage    schema.ResourceUsage
    Region   string
    Q        PriceQuerier
}
type PricingOutput struct {
    Components   []schema.CostComponent
    NotEstimated []schema.NotEstimated
}
type Pricer interface {
    ResourceTypes() []string
    Price(ctx context.Context, in PricingInput) (PricingOutput, error)
}
```

- `Price` returns an `error` only for a genuine contract fault (bad usage of `Q`, a panic-worthy state). "Can't price X" is a `NotEstimated` entry.
- A pricer emits `CostComponent` for every priceable piece and `NotEstimated{reason}` for pieces it cannot (missing attr → `MISSING_ATTRIBUTE`; unknown-after-apply → `UNKNOWN_AFTER_APPLY`; usage-based with no usage value → `NO_USAGE_DATA`; unmodelled sub-config → `UNSUPPORTED_CONFIGURATION`; query error → `PRICING_API_ERROR`).
- Helper `pricing.HourlyToMonthly(usd string) string`, `pricing.Component(...)` builder centralises `big.Rat` math and 730h conversion (rounding half-away-from-zero, 10dp price / 4dp cost, per U0).

### 1.4 Catalog (`internal/pricing`, Q5=B explicit constructor)

```go
func NewCatalog() Catalog        // returns all v1 pricers, no init()
type Catalog interface {
    For(resourceType string) (Pricer, bool)
    All() []Pricer
}
```

v1 pricers (`internal/pricing/aws`):

| File | Pricers | Terraform types |
|---|---|---|
| `compute.go` | instance, ebsVolume, ebsSnapshot, eip | `aws_instance`, `aws_ebs_volume`, `aws_ebs_snapshot`(+`_copy`), `aws_eip` |
| `database.go` | rdsInstance, aurora, elastiCache, dynamoDB | `aws_db_instance`, `aws_rds_cluster`(+`_instance`), `aws_elasticache_cluster`(+`_replication_group`), `aws_dynamodb_table` |
| `network.go` | loadBalancer, natGateway, dataTransfer | `aws_lb`/`aws_alb`/`aws_elb`, `aws_nat_gateway`, (`data_transfer` synthetic, usage-only) |
| `serverless.go` | lambda | `aws_lambda_function` |
| `storage.go` | s3 | `aws_s3_bucket` |
| `containers.go` | eksCluster, eksNodeGroup | `aws_eks_cluster`, `aws_eks_node_group` |
| `observability.go` | cloudWatch | `aws_cloudwatch_log_group`, `aws_cloudwatch_metric_alarm`, `aws_cloudwatch_dashboard` |

### 1.5 RegionResolver (`internal/pricing`)

```go
func (r RegionResolver) Resolve(res plan.Resource, cfgs map[string]plan.ProviderConfig, override string) (string, bool)
```

Order: `override` (from `--aws-region`) > `cfgs[res.ProviderConfigKey].ConstantRegion` > `("", false)` → caller records `MISSING_ATTRIBUTE`.

### 1.6 PriceListClient (`internal/pricing`) — US-PRICING-1/6, NFR-1.2

- Wraps `pricing.Client` from `aws-sdk-go-v2/service/pricing`.
- `Query`:
  - **dedup**: `golang.org/x/sync/singleflight` keyed by the canonical query key; plus a per-run `map[string][]PriceDimension` memo (mutex-guarded) so a completed query is never re-issued.
  - **worker pool**: a buffered semaphore channel, capacity = configured concurrency (default 8); `Query` acquires before calling the SDK.
  - **backoff**: SDK's adaptive retryer configured (`retry.AddWithMaxAttempts`, `retry.AddWithErrorCodes` for `ThrottlingException`); on final failure `Query` returns a wrapped error.
  - **pagination**: follows `NextToken` from `GetProducts`.
  - **stats**: `QueryStats{Issued, DedupHits, ThrottleRetries}`.
- Endpoint: `pricing` service is only available in `us-east-1` and `ap-south-1`; `awsauth` picks one. The queried region goes in the `regionCode` filter, not the endpoint.

### 1.7 CachingClient + CacheStore (`internal/pricing`) — US-CACHE-1/2/3, FR-4

- `CacheKey(q PriceQuery)`: SHA-256 of a canonical string `servicecode|regioncode|sorted(field=value;...)` — `Purpose` excluded.
- `CacheEntry{StoredAt time.Time; Dimensions []PriceDimension}` stored as JSON, gzip optional (skip for v1).
- Disk layout: `<cacheDir>/v1/<first2>/<sha>.json`; `<cacheDir>` default `~/.price-checker/cache`, override `--cache-dir` / `PRICE_CHECKER_CACHE_DIR`.
- TTL default 168h (7d); `--cache-ttl`. Stale entry → refetch + overwrite.
- `--no-cache`: bypass read and write. `--refresh-cache`: skip read, do write.
- Atomic write: temp file in same dir + `os.Rename`; dir `0700`, file `0600`.
- Corrupt/મunreadable entry → treated as miss (logged at debug), refetched.
- Concurrency-safe: rename is atomic; readers tolerate a missing/partial file.
- Never stores plan-derived data — only the query key and the Price List response.
- Stats merged: `CacheHits`, `CacheMisses` → `CacheHitRatio`.

### 1.8 Price List response parsing (`internal/pricing/pricelist.go`)

`GetProducts` returns `PriceList []string`, each a JSON doc:
```
{ "product": { "attributes": { "instanceType": "t3.medium", ... } },
  "terms": { "OnDemand": { "<sku>": { "priceDimensions": {
      "<ratecode>": { "unit": "Hrs", "description": "...", "pricePerUnit": { "USD": "0.0416000000" } } } } } } }
```
Parser extracts one `PriceDimension` per on-demand price dimension; ignores `Reserved`. `USD` kept as the raw decimal string. If `USD` absent/empty → dimension skipped.

### 1.9 `awsauth` (`internal/awsauth`)

```go
func Load(ctx context.Context, profile, regionOverride string) (aws.Config, error)
func PriceListRegion(cfg aws.Config) string   // "us-east-1" unless caller in ap-south-1
```
- Uses `config.LoadDefaultConfig` (env, shared config/credentials, SSO, container/instance roles).
- On missing credentials, `Load` still succeeds (lazy); the **first** `GetProducts` call surfaces the auth error. `PriceListClient` wraps that into an actionable message: "AWS credentials not found or invalid — configure via environment, shared config, a named profile (--profile), or an instance role; see --help".
- `--profile` → `config.WithSharedConfigProfile`.

---

## 2. NFR Requirements (unit-scoped)

| NFR | Obligation |
|---|---|
| NFR-1.1/1.2 | worker pool (default 8) + dedup; warm cache = 0 queries |
| NFR-2.1 | all money math via `pkg/schema` helpers (`big.Rat`) |
| NFR-3.2/3.4 | cache stores no plan data; `0600`/`0700` perms |
| NFR-5.1 | new resource type = implement `Pricer` + one line in `catalog.go` |
| NFR-5.2 | every pricer has recorded `GetProducts` fixtures; a fake `PriceQuerier` serves them in tests |
| NFR-5.3 | `internal/pricing/aws` must NOT import `aws-sdk-go-v2` (import-lint test) |
| NFR-7.1/7.2 | debug logs per query + cache hit/miss; `QueryStats` feeds `RunMetadata` |

### 2.1 Dependencies (added to go.mod)

| Import | Scope | Purpose |
|---|---|---|
| `github.com/aws/aws-sdk-go-v2` + `/config` + `/service/pricing` | non-test (`awsauth`, `internal/pricing`) | Price List API |
| `golang.org/x/sync/singleflight` | non-test (`internal/pricing`) | query dedup |

`internal/pricing/aws` and downstream units gain **no** new deps.

---

## 3. NFR Design

- **Interface seam for tests**: `PriceListClient` depends on a minimal `getProductsAPI` interface (`GetProducts(ctx, *pricing.GetProductsInput, ...) (*pricing.GetProductsOutput, error)`); tests inject a fake, no network.
- **`PriceQuerier` is the only thing pricers see** — a fake implementing `Query` from JSON fixtures drives every pricer test.
- **Deterministic dimension order**: parser sorts `[]PriceDimension` by `(SKU, ratecode)` so cache writes and pricer selection are stable.
- **Cache is oblivious to schema**: stores `PriceDimension` (its own type), not `schema` types.
- **Worker pool lifecycle**: created by `NewPriceListClient`, closed by `Close()`; `Estimator` (U3) owns the lifecycle.
- **Rounding**: centralised in `pricing.Component()` and `pricing.HourlyToMonthly()`; nowhere else.

### 3.1 File plan

```
internal/awsauth/awsauth.go           Load, PriceListRegion
internal/pricing/doc.go
internal/pricing/types.go             PriceQuery, Filter, PriceDimension, QueryStats, canonical key
internal/pricing/pricelist.go         parse GetProducts PriceList JSON -> []PriceDimension
internal/pricing/client.go            PriceListClient (pool, singleflight, memo, backoff, stats)
internal/pricing/cache.go             CachingClient + diskStore + CacheOptions
internal/pricing/pricer.go            Pricer, PricingInput/Output, PriceQuerier, Component helpers
internal/pricing/region.go            RegionResolver
internal/pricing/catalog.go           NewCatalog() -> all aws pricers
internal/pricing/aws/*.go             pricer implementations (7 files, ~16 types)
internal/pricing/aws/common.go        shared helpers (region filter, attr extraction)
internal/pricing/aws/testdata/*.json  recorded GetProducts PriceList entries
*_test.go                             per layer + per pricer
```

### 3.2 Test targets (DoD)

- `pricelist_test.go`: parse a recorded `GetProducts` doc → expected dimensions; skip `Reserved`; skip empty USD.
- `client_test.go`: fake `getProductsAPI`; dedup (2 identical concurrent queries → 1 SDK call, `DedupHits==1`); pagination; throttle→retry→success; final failure → wrapped error.
- `cache_test.go`: miss→store→hit; TTL expiry; `--no-cache`; `--refresh-cache`; corrupt file → miss; two goroutines writing same key both succeed; file perms `0600`.
- `region_test.go`: override wins; provider-config constant; unresolved → `("",false)`.
- `catalog_test.go`: `For` covers every documented type; `All()` non-empty; no duplicate type registration.
- `aws/*_test.go`: each pricer against fixtures — happy path components; missing attr → `MISSING_ATTRIBUTE`; unknown-after-apply → `UNKNOWN_AFTER_APPLY`; usage-based w/o usage → `NO_USAGE_DATA`.
- **PBT** (`rapid`): `pricing.Component()` — monthly == hourly*730 within rounding; non-negative; currency USD (PBT-03).
- `imports_test.go` (x2): `internal/pricing/aws` has no `aws-sdk-go-v2`; `internal/pricing` has no `internal/plan`-forbidden edges? (it may import plan — allowed).

---

## 4. Open items → later units

- Exact filter sets are encoded per-pricer with a fixture; refinements land as fixture + pricer edits (no design change).
- Data-transfer pricing is usage-file-only in v1 (`data_transfer` synthetic resource not emitted by the plan; folded into NAT/other as usage keys) — full DTO modelling deferred.
- `--concurrency`, `--cache-dir`, `--cache-ttl`, `--no-cache`, `--refresh-cache` flags are wired in U5; U2 exposes them as `CacheOptions` / client config.
