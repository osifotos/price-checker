# U2 `pricing-core` — Code Summary

**Locations**: `internal/awsauth/`, `internal/pricing/`, `internal/pricing/aws/`

> Go toolchain absent in this environment. `go mod tidy` MUST be run in Build and Test to populate `go.sum` and pin transitive AWS SDK modules (credentials, smithy-go, etc.). Versions in `go.mod` are indicative and may be bumped by `go mod tidy`.

## New dependencies (go.mod)

| Module | Scope |
|---|---|
| `github.com/aws/aws-sdk-go-v2` + `/config` + `/service/pricing` | non-test — only `internal/awsauth` and `internal/pricing` |
| `golang.org/x/sync` (`singleflight`) | non-test — `internal/pricing` |

`internal/pricing/aws` imports **none** of these (enforced by `imports_test.go`).

## Files

### `internal/awsauth/`
| File | Contents |
|---|---|
| `awsauth.go` | `Load` (SDK default credential chain, `--profile`, region override), `PriceListRegion` (us-east-1 / ap-south-1 only), `CredentialHelp` |

### `internal/pricing/`
| File | Contents |
|---|---|
| `doc.go` | two-layer architecture doc |
| `types.go` | `PriceQuery`, `Filter`, `PriceDimension`, `QueryStats` (+ `CacheHitRatio`), `PriceQuerier`, canonical SHA-256 query key (Purpose excluded) |
| `pricelist.go` | parse `GetProducts` PriceList JSON → deterministically ordered `[]PriceDimension`; skips Reserved terms and empty-USD dimensions |
| `client.go` | `PriceListClient`: bounded worker pool (default 8), `singleflight` + in-memory memo dedup, pagination, credential-error wrapping, `QueryStats`, `Close` |
| `cache.go` | `CachingClient` decorator + `diskStore`: TTL (default 7d), `--no-cache` / `--refresh-cache`, atomic temp+rename writes, `0700`/`0600`, corrupt entry = miss, `$PRICE_CHECKER_CACHE_DIR` / `~/.price-checker/cache` |
| `pricer.go` | `Pricer`, `PricingInput`/`PricingOutput` (+ `Add`/`Skip`), money helpers `Component` / `HourlyComponent` / `SumMonthly` / `SumHourly` (all `big.Rat`, 730h, half-away-from-zero) — the only place cost math happens |
| `region.go` | `RegionResolver`: override > provider-config constant > unresolved |
| `catalog.go` | `NewCatalog(pricers ...Pricer)` — explicit, no `init()` |

### `internal/pricing/aws/` (pricers — no AWS SDK import)
| File | Pricers |
|---|---|
| `common.go` | `Pricers()`, `NewCatalog()`, `query1` (cheapest dimension), `f` filter helper |
| `compute.go` | `aws_instance` (+ block devices), `aws_ebs_volume` (+ IOPS), `aws_ebs_snapshot`/`_copy`, `aws_eip` |
| `database.go` | `aws_db_instance` (+ storage), `aws_rds_cluster`/`_instance` (Aurora), `aws_elasticache_cluster`/`_replication_group`, `aws_dynamodb_table` |
| `network.go` | `aws_lb`/`aws_alb`/`aws_elb`, `aws_nat_gateway` (+ data processed) |
| `serverless.go` | `aws_lambda_function` (requests + GB-seconds) |
| `storage.go` | `aws_s3_bucket` (storage + tier1/tier2 requests) |
| `containers.go` | `aws_eks_cluster`, `aws_eks_node_group` |
| `observability.go` | `aws_cloudwatch_log_group` / `_metric_alarm` / `_dashboard` |

All 22 Terraform resource types from the v1 catalog are registered (Q5=A).

## Tests

| File | Covers |
|---|---|
| `awsauth_test.go` | Price List region selection; credential help present |
| `pricelist_test.go` | parse; Reserved + zero-price skipped; deterministic SKU order |
| `client_test.go` | **dedup** (10 concurrent identical queries → 1 SDK call, ≥9 dedup hits); pagination; error wrapped |
| `cache_test.go` | miss→hit; TTL expiry; `--no-cache`; `--refresh-cache`; corrupt=miss; concurrent writers; file perms `0600` |
| `region_test.go` | override wins; provider constant; aliased; unresolved |
| `aws/catalog_test.go` | every documented type has a pricer; `aws_kms_key` not claimed; no duplicate registration |
| `aws/pricer_test.go` | instance (happy + unknown-after-apply + API-error reason), EBS volume, snapshot (NO_USAGE_DATA), NAT gateway (with/without usage), RDS (multi-az + storage), Lambda (needs usage) — via fake `PriceQuerier` |
| `component_pbt_test.go` | **PBT-03** (`rapid`): hourly×730 ≈ monthly, non-negative, price echoed; `SumMonthly` |
| `aws/imports_test.go` | **NFR-5.1/5.3** pricers never import the AWS SDK |

## Stories advanced

- **US-PRICING-1** (live Price List), **US-PRICING-2** (credential chain + actionable error), **US-PRICING-3** (RegionResolver), **US-PRICING-4** (22-type catalog), **US-PRICING-5** (reason codes emitted by every pricer), **US-PRICING-6** (worker pool + dedup + backoff + partial-failure `PRICING_API_ERROR`), **US-PRICING-7** (one-method `Pricer` + explicit catalog).
- **US-CACHE-1/2/3** (disk cache, TTL/flags, atomic + corrupt-tolerant + perms).

## Notes / deviations

- Price List **filter fields** (e.g. `usagetype` prefixes, `group`, `productFamily`) are best-effort from public docs; several pricers fall back to a coarser query if the tight filter misses. Recorded real fixtures + refinement are a follow-on within U2's DoD (tracked; needs a machine with AWS access) — the fake-`PriceQuerier` tests fully exercise the pricer logic regardless.
- Data-transfer is modelled only as usage keys on NAT/other resources in v1; no standalone `data_transfer` pricer.
- `aws.NewCatalog()` is the entry point U3's Estimator will call.
