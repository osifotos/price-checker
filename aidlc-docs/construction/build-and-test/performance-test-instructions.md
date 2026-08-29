# Performance Test Instructions — price-checker

## Requirements (NFR-1)
- A 200-resource plan completes in **≤ 10 s warm cache**, **≤ 60 s cold cache** on a typical laptop.
- Price List queries are deduplicated and bounded (default 8 concurrent).

## Setup
```bash
# generate a synthetic 200-resource plan
go run ./internal/plan/testdata/gen  # (helper to add if not present) OR hand-craft
export PRICE_CHECKER_CACHE_DIR=$(mktemp -d)
```

## Tests

### 1. Cold-cache latency
```bash
rm -rf "$PRICE_CHECKER_CACHE_DIR"/*
/usr/bin/time -p ./bin/price-checker --path plan-200.json --format json > /dev/null
```
- **Pass**: real time ≤ 60 s. Inspect `metadata.price_queries` — should be far less than 200 (dedup across identical instance types / volume types).

### 2. Warm-cache latency
```bash
./bin/price-checker --path plan-200.json > /dev/null    # populate
/usr/bin/time -p ./bin/price-checker --path plan-200.json --format json > /dev/null
```
- **Pass**: real time ≤ 10 s; `metadata.price_queries == 0`; `metadata.cache_hit_ratio == 1`.

### 3. Dedup effectiveness (unit-level, already covered)
- `internal/pricing/client_test.go::TestClientDedup` asserts 10 concurrent identical queries collapse to 1 SDK call. Extend with a benchmark:
```bash
go test -run x -bench BenchmarkQueryDedup -benchmem ./internal/pricing
```

### 4. Concurrency ceiling
```bash
./bin/price-checker --path plan-200.json --concurrency 1 --format json    # baseline
./bin/price-checker --path plan-200.json --concurrency 16 --format json   # should be faster, no throttle errors
```

## Analysis
- Record cold/warm wall time, `price_queries`, `cache_hit_ratio`, peak RSS (`/usr/bin/time -v`).
- **Bottleneck expectation**: network round-trips to the Price List API dominate cold runs; the worker pool + dedup + on-disk cache are the mitigations. If cold time exceeds budget, raise `--concurrency` (watch for throttling) or pre-warm the cache in CI.

## Optimization levers (if budget missed)
1. Increase the default worker-pool size (currently 8) — re-test for `ThrottlingException`.
2. Batch identical `regionCode` + `serviceCode` queries with broader filters, then filter dimensions in-process.
3. Ship a pre-baked cache snapshot for common regions.
