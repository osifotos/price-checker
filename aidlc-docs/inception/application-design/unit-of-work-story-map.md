# Unit of Work → Story Map — Price Checker

Every one of the 30 stories has **exactly one owning unit** (where its acceptance criteria are primarily satisfied) and, where relevant, **supporting units** that contribute.

## By story

| Story | Owning unit | Supporting units | Tag |
|---|---|---|---|
| US-LOCAL-ESTIMATE-1 — provide plan (file/stdin) | U1 plan-ingest | U0 | v1 |
| US-LOCAL-ESTIMATE-2 — price a Terraform directory | U1 plan-ingest | U5 (flag) | v1 |
| US-LOCAL-ESTIMATE-3 — read a cost breakdown table | U4 output-and-diff | U3 (totals), U5 (flags) | v1 |
| US-LOCAL-ESTIMATE-4 — components + period | U4 output-and-diff | U3 (big.Rat, 730h), U5 (`--period`) | v1 |
| US-PRICING-1 — live Price List prices | U2 pricing-core | U0 | v1 |
| US-PRICING-2 — credentials + clear failure | U2 pricing-core (`awsauth`) | U5 (error surface) | v1 |
| US-PRICING-3 — per-resource region + override | U2 pricing-core (`RegionResolver`) | U1 (provider config), U5 (`--aws-region`) | v1 |
| US-PRICING-4 — cover core AWS cost drivers | U2 pricing-core (`aws` pricers) | U1 | v1 |
| US-PRICING-5 — not-estimated + reason codes | U3 cost-engine (Aggregator collects) | U2 (emits), U0 (`ReasonCode`), U4 (renders) | v1 |
| US-PRICING-6 — resilient to API throttling | U2 pricing-core (`PriceListClient`) | U3 (Estimator loop) | v1 |
| US-PRICING-7 — add a resource type via one interface | U2 pricing-core (`Pricer`/`Catalog`) | U0, U4 (renderer isolation) | v1 |
| US-CACHE-1 — cache pricing responses on disk | U2 pricing-core (`CachingClient`/`CacheStore`) | U5 (`--cache-dir`) | v1 |
| US-CACHE-2 — control cache freshness | U2 pricing-core | U5 (`--cache-ttl`,`--no-cache`,`--refresh-cache`), U3 (stats in metadata) | v1 |
| US-CACHE-3 — safe under concurrency/corruption | U2 pricing-core (`CacheStore`) | — | v1 |
| US-USAGE-1 — supply usage assumptions | U3 cost-engine (`usage`) | U0 (`UsageFile`), U5 (`--usage-file`) | v1 |
| US-USAGE-2 — warnings not failures | U3 cost-engine (`usage`) | U5 (stderr) | v1 |
| US-USAGE-3 — scaffold a usage file | U3 cost-engine (`UsageSkeletonGenerator`) | U2 (`Catalog.All`), U5 (`usage generate` cmd) | v1 |
| US-MACHINE-OUTPUT-1 — stable versioned JSON | U0 schema | U4 (JSONRenderer), U3 (metadata) | v1 |
| US-MACHINE-OUTPUT-2 — write output to a file | U4 output-and-diff (`--out` helper) | U5 (flag) | v1 |
| US-REPORT-1 — self-contained HTML report | U4 output-and-diff (`render/html`) | U5 (`report` cmd) | v1 |
| US-DIFF-1 — compare two cost states | U4 output-and-diff (`Differ`) | U3 (`EstimateBothStates`), U5 (`InputResolver`) | v1 |
| US-DIFF-2 — diff in every format | U4 output-and-diff (diff renderers) | U0 (diff schema) | v1 |
| US-CI-GATE-1 — predictable exit codes | U5 cli-app (`ExitPolicy`) | — | v1 |
| US-CI-GATE-2 — threshold gates + strict | U5 cli-app (`ExitPolicy`) | U3 (totals), U2 (not-estimated for `--strict`) | v1 |
| US-PR-COMMENT-1 — emit a PR comment body | U4 output-and-diff (`GitHubCommentRenderer`) | U5 (`--format github-comment`) | v1 |
| US-PR-COMMENT-2 — understand change from comment | U4 output-and-diff (`GitHubCommentRenderer` + `Differ`) | — | v1 |
| US-PR-COMMENT-3 — CI wiring recipe | U5 cli-app (README/docs) | — | later |
| US-CONFIG-INSTALL-1 — team-wide defaults via config file | U5 cli-app (`config`) | — | v1 |
| US-CONFIG-INSTALL-2 — discoverable CLI, clean streams | U5 cli-app (`cli`, `Logging`) | U2 (debug: query logs), U2/U5 (cache hit/miss logs) | v1 |
| US-CONFIG-INSTALL-3 — single binary, 5 platforms | U5 cli-app (`.github/workflows`, `cmd`) | all (no-cgo discipline) | v1 |

## By unit (coverage roll-up)

| Unit | Owns (primary) | Supports |
|---|---|---|
| **U0 schema** | US-MACHINE-OUTPUT-1 | US-LOCAL-ESTIMATE-1, US-PRICING-1/5/7, US-USAGE-1, US-MACHINE-OUTPUT-2, US-DIFF-2 |
| **U1 plan-ingest** | US-LOCAL-ESTIMATE-1, US-LOCAL-ESTIMATE-2 | US-PRICING-3/4, US-DIFF-1 |
| **U2 pricing-core** | US-PRICING-1/2/3/4/6/7, US-CACHE-1/2/3 | US-PRICING-5, US-USAGE-3, US-CI-GATE-2, US-CONFIG-INSTALL-2 |
| **U3 cost-engine** | US-LOCAL-ESTIMATE-3*/4*, US-PRICING-5, US-USAGE-1/2/3 | US-CACHE-2, US-DIFF-1, US-CI-GATE-2, US-MACHINE-OUTPUT-1 |
| **U4 output-and-diff** | US-LOCAL-ESTIMATE-3/4, US-MACHINE-OUTPUT-2, US-REPORT-1, US-DIFF-1/2, US-PR-COMMENT-1/2 | US-PRICING-5/7 |
| **U5 cli-app** | US-CI-GATE-1/2, US-CONFIG-INSTALL-1/2/3, US-PR-COMMENT-3 (later) | US-LOCAL-ESTIMATE-2, US-PRICING-2/3, US-CACHE-1/2, US-USAGE-1/3, US-MACHINE-OUTPUT-2, US-REPORT-1, US-DIFF-1 |

\* US-LOCAL-ESTIMATE-3/4 acceptance criteria split: numeric/aggregation behaviour is U3; the rendered table/components view is U4. Listed as owned by U4 in the per-story table (where the user-visible behaviour lands) with U3 support.

## Coverage check

- **30 / 30 stories assigned** to an owning unit.
- **29 v1** stories land within U0–U5; **1 `later`** story (US-PR-COMMENT-3) is docs in U5.
- No story is owned by two units. Every unit owns ≥ 1 story.
- Cross-cutting NFR acceptance criteria (attached to stories per clarification 1 = A) travel with their host story: PBT → US-MACHINE-OUTPUT-1 (U0), extensibility → US-PRICING-7 (U2), portability → US-CONFIG-INSTALL-3 (U5).
