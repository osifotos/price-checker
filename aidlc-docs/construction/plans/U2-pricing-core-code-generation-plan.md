# U2 `pricing-core` — Code Generation Plan (consolidated cadence)

Code: `internal/awsauth/`, `internal/pricing/`, `internal/pricing/aws/`.
Stories: US-PRICING-1..7, US-CACHE-1..3.

| # | Step | Status |
|---|---|---|
| 1 | go.mod: add aws-sdk-go-v2 (config, service/pricing), golang.org/x/sync | [x] |
| 2 | `internal/awsauth/awsauth.go` | [x] |
| 3 | `internal/pricing/types.go` + `pricelist.go` | [x] |
| 4 | `internal/pricing/client.go` (pool, singleflight dedup, pagination, backoff, stats) | [x] |
| 5 | `internal/pricing/cache.go` (CachingClient + diskStore) | [x] |
| 6 | `internal/pricing/pricer.go` (Pricer, IO types, money helpers) | [x] |
| 7 | `internal/pricing/region.go` + `catalog.go` | [x] |
| 8 | `internal/pricing/aws/*.go` — 22 resource types across 8 files | [x] |
| 9 | Tests: awsauth, pricelist, client (dedup/pagination), cache, region, catalog, per-pricer (fake querier), component PBT, import-lint | [x] |
| 10 | `code-summary.md` | [x] |

Follow-on within U2 DoD (needs AWS access): record real `GetProducts` fixtures and tighten filter sets.
Build/lint/test + `go mod tidy` deferred to Build and Test stage.
