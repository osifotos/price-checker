# Integration Test Instructions — price-checker

The units are packages in one binary; "integration" here means (a) the wired
pipeline across `plan → pricing → engine → render`, and (b) real AWS Price List
API behaviour.

## A. Offline cross-unit integration (no AWS)

These run in `go test` today via `internal/cli/run_test.go` and
`internal/estimator/estimator_test.go` with a stubbed querier. To extend:

### Scenario 1 — plan JSON → table (end to end, stubbed prices)
- **Setup**: a committed plan fixture (`internal/plan/testdata/ec2_s3_natgw.json`) and a fake `pricing.PriceQuerier` returning recorded dimensions.
- **Steps**: `cli.runWithDeps(["price-checker","--path","<fixture>"], …, deps)` with `deps.NewEstimator` returning an `estimator.New` wired to the fake querier + real `aws.NewCatalog()`.
- **Expected**: exit 0; table lists every managed resource; not-estimated section lists `aws_kms_key` (`UNSUPPORTED_TYPE`) and usage-based components (`NO_USAGE_DATA`).

### Scenario 2 — diff over two plan states
- **Steps**: `price-checker diff --from-plan --path <replace-plan-fixture>`.
- **Expected**: `changes[]` shows the changed resource; `total_delta_monthly` = new − prior.

### Scenario 3 — JSON contract
- **Steps**: `price-checker --path <fixture> --format json | <validator>`.
- **Expected**: output validates against `pkg/schema/breakdown.schema.json` (already asserted in `render_test.go`).

### Scenario 4 — CI gates
- **Steps**: run with `--strict` against a fixture containing an unsupported type; run with `--threshold-monthly 0.01`.
- **Expected**: exit 2 and exit 3 respectively; full output still printed.

## B. Live AWS Price List integration (opt-in, needs credentials)

Guarded behind a build tag / env so normal `go test` never hits the network.

### Setup
```bash
export PRICE_CHECKER_LIVE=1
aws sts get-caller-identity        # confirm credentials with pricing:GetProducts
```

### Test: real prices for the catalog
- For each catalogued resource type, build a minimal synthetic `plan.Resource`, call its pricer through a **real** `pricing.NewCachingClient(pricing.NewPriceListClient(awspricing.NewFromConfig(cfg), …))`, and assert:
  - at least one `CostComponent` is produced (or a documented `NotEstimated` reason for usage-only types),
  - `price_per_unit` parses as a positive decimal,
  - the value is within a sanity band of the public AWS pricing page (±20%).
- **Record fixtures**: on success, persist the raw `GetProducts` responses under `internal/pricing/aws/testdata/<type>.json` so the offline per-pricer tests gain real data (this closes the U2 filter-accuracy follow-on).

### Test: cache + concurrency against the real API
- Run a 50-resource plan twice; assert run 2 issues **0** Price List queries (`RunMetadata.price_queries == 0`, `cache_hit_ratio == 1`).
- Run with `--concurrency 8` and confirm no `ThrottlingException` surfaces as a hard error.

### Cleanup
```bash
rm -rf ~/.price-checker/cache        # or use --cache-dir $(mktemp -d)
```

## C. Directory-input integration (needs terraform)
- **Setup**: a tiny real Terraform module (1 `aws_instance`), `terraform init`.
- **Steps**: `price-checker --path ./that-module`.
- **Expected**: the tool runs `terraform plan`/`show -json`, prices the instance, cleans up its temp plan file.
