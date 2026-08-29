# Requirements — Price Checker (Terraform Cloud Cost Calculator)

## Intent Analysis Summary

- **User request**: "build a price calculator. think of infracost" — a CLI tool that estimates the cloud cost of infrastructure described by Terraform, following the AI-DLC workflow.
- **Request type**: New Project (greenfield)
- **Scope estimate**: Multiple Components (parser, pricing client, cost engine, usage model, multiple output renderers, CLI, CI integration)
- **Complexity estimate**: Moderate–Complex
- **Requirements depth**: Standard
- **Reference product**: Infracost (behavioural reference, not a dependency)

## Settled Decisions (from clarification rounds)

| Topic | Decision |
|---|---|
| Input | Terraform plan JSON; from `--path <file>`, stdin, **or** by running `terraform show -json` against a given Terraform directory (Q1:B) |
| Provider scope | AWS only for v1 (AWS core services) |
| Pricing data source | Live AWS Price List Query API, USD |
| Language | Go |
| Output formats | CLI table, JSON, diff view, HTML report (Q2:D) |
| Pricing catalog | Fixed v1 catalog of common AWS cost drivers; everything else → "not estimated" (Q3 → clarification 1a:A) |
| Coverage reporting | Always list in-plan resources that were not priced, with a reason (clarification 1b:A) |
| Usage-based costs | Optional usage file supplies monthly usage assumptions (Q4:B); no built-in defaults |
| Credentials / region | Standard AWS SDK credential chain; region from plan/provider config, overridable by `--aws-region` (Q5:A) |
| Caching | On-disk cache with configurable TTL; `--no-cache` and `--refresh-cache` flags (Q6:B) |
| Unpriceable resources | "Not estimated" section with reason, exit 0; `--strict` makes it exit non-zero (Q7:C) |
| Use context | Local CLI + CI pipeline (stable JSON, exit codes, cost-threshold gate) + PR cost-summary comment mode (Q8:C) |
| Cost period / currency | Monthly (730h) and hourly, **USD only** for v1, `--period hour|month` flag; configurable currency deferred (Q9 → clarification 2:A) |

## Extension Configuration

| Extension | Enabled | Notes |
|---|---|---|
| Security Baseline | No | User opted out (treated as prototype/experimental) |
| Resiliency Baseline | No | User opted out |
| Property-Based Testing | Yes — **Partial** | Enforced rules: PBT-02, PBT-03, PBT-07, PBT-08, PBT-09 (round-trip, invariant, generator quality, shrinking/reproducibility, framework selection). Others advisory. |

---

## Functional Requirements

### FR-1 — Terraform plan ingestion
- FR-1.1 Accept a Terraform plan in JSON format (output of `terraform show -json`) via a `--path <file>` argument.
- FR-1.2 Accept the same JSON on stdin when no path is given or `--path -` is passed.
- FR-1.3 When `--path` points to a directory containing Terraform configuration, run `terraform show -json` (and `terraform plan` as needed to produce a plan file) against it and consume the result. Require the `terraform` binary on PATH for this mode; produce a clear error if absent.
- FR-1.4 Support Terraform plan JSON format versions produced by Terraform 1.x (`format_version` 0.1 and 1.x families). Reject unknown/older formats with a clear message.
- FR-1.5 Extract planned resources from `planned_values` / `resource_changes`, including resource type, name, address, module path, and resolved attribute values. Resources marked for deletion contribute zero cost in single-plan mode.

### FR-2 — AWS resource pricing catalog (v1 fixed catalog)
The following AWS resource types MUST receive real cost estimates in v1. Each entry lists the primary cost components modelled.

| Terraform resource type | Cost components modelled |
|---|---|
| `aws_instance` | On-demand instance hours (by instance type, region, OS/tenancy); attached root + EBS block device storage; detailed monitoring (usage-file optional) |
| `aws_ebs_volume` | Provisioned storage GB-month by volume type; provisioned IOPS (io1/io2/gp3); provisioned throughput (gp3) |
| `aws_ebs_snapshot` / `aws_ebs_snapshot_copy` | Snapshot storage GB-month (usage-file for size) |
| `aws_eip` | Idle/allocated Elastic IP hours per current AWS pricing rules |
| `aws_db_instance` | RDS instance hours (engine, class, single/multi-AZ); allocated storage GB-month; provisioned IOPS; backup storage (usage-file) |
| `aws_rds_cluster` / `aws_rds_cluster_instance` (Aurora) | Aurora instance hours; storage and I/O marked usage-based |
| `aws_s3_bucket` | Storage GB-month, request counts, data retrieval — all usage-based via usage file |
| `aws_lambda_function` | Request count and GB-seconds (memory × duration) — usage-based via usage file; provisioned concurrency if set |
| `aws_lb` / `aws_alb` / `aws_elb` | Load balancer hours by type (ALB/NLB/GLB/classic); LCU / capacity-unit hours usage-based via usage file |
| `aws_nat_gateway` | NAT Gateway hours; GB data processed usage-based via usage file |
| `aws_eks_cluster` | Cluster (control plane) hours |
| `aws_eks_node_group` | Underlying EC2 instance hours per node × desired size |
| `aws_elasticache_cluster` / `aws_elasticache_replication_group` | Node hours by node type and engine × node count |
| `aws_dynamodb_table` | Provisioned RCU/WCU hours when billing mode is PROVISIONED; storage and on-demand request costs usage-based |
| `aws_cloudwatch_*` (log group, metric alarm, dashboard) | Alarm/dashboard fixed monthly costs; log ingestion/storage usage-based |
| Inter-region / internet data transfer | Usage-based via usage file, keyed by region |

- FR-2.1 For each catalogued resource, compute a monthly cost and, where the underlying unit is hourly, an hourly cost.
- FR-2.2 Resource attributes that are unknown at plan time (`(known after apply)`) MUST be handled: fall back to usage-file value or mark the affected component "not estimated" with the reason.
- FR-2.3 `count` and `for_each` expansions MUST be priced per instance.

### FR-3 — Pricing data retrieval
- FR-3.1 Retrieve prices from the AWS Price List Query API (`GetProducts` / `GetAttributeValues`) using the AWS SDK for Go v2.
- FR-3.2 Resolve credentials via the standard AWS SDK credential chain (environment, shared config/credentials files, named profiles, SSO, container/instance roles).
- FR-3.3 Determine the AWS region per resource from the plan's provider configuration; allow a global override via `--aws-region`. Price List API calls are made against the `us-east-1` / `ap-south-1` Price List endpoints regardless of the priced region.
- FR-3.4 Map each catalogued resource + attribute set to a Price List service code and filter set, and select the correct price dimension (on-demand, the relevant term and rate).
- FR-3.5 On an API error (auth, throttling, network), retry with backoff; if still failing, mark affected resources "not estimated" with the reason, unless `--strict` is set (then exit non-zero).

### FR-4 — Pricing cache
- FR-4.1 Cache Price List query responses on disk (default `~/.price-checker/cache`, overridable via `--cache-dir` or `PRICE_CHECKER_CACHE_DIR`).
- FR-4.2 Cache entries carry a timestamp; entries older than the TTL (default 7 days, overridable via `--cache-ttl`) are treated as stale and refetched.
- FR-4.3 `--no-cache` bypasses read and write. `--refresh-cache` ignores existing entries on read but writes fresh results.
- FR-4.4 Cache must be safe for concurrent runs (atomic writes; tolerate corrupt entries by refetching).

### FR-5 — Usage file
- FR-5.1 Accept an optional usage file via `--usage-file <path>` in YAML, modelled on Infracost's `infracost-usage.yml` structure (`version`, `resource_usage` keyed by resource address, with per-component monthly quantities).
- FR-5.2 Usage keys map to the usage-based components listed in FR-2 (e.g. `monthly_data_processed_gb` for NAT Gateway, `monthly_requests` and `request_duration_ms`/`memory` for Lambda, `storage_gb` for S3).
- FR-5.3 Components with no usage value provided remain "not estimated" (no built-in default assumptions in v1).
- FR-5.4 Validate the usage file; report unknown resource addresses and unknown keys as warnings, not fatal errors.
- FR-5.5 Provide a `price-checker usage generate` subcommand that emits a skeleton usage file listing every usage-based component found in the plan with empty values and inline comments.

### FR-6 — Cost calculation engine
- FR-6.1 Produce a structured cost breakdown: per resource → list of cost components → (unit, price, monthly quantity, monthly cost, hourly cost).
- FR-6.2 Aggregate totals per module and a project grand total (monthly and hourly).
- FR-6.3 Separate "estimated" totals from "not estimated" items; the grand total reflects only estimated components and states how many components/resources were not estimated.
- FR-6.4 All monetary math uses a decimal type (no binary float accumulation); monthly hours constant = 730.
- FR-6.5 Deterministic output ordering (by resource address, then component name).

### FR-7 — Output: CLI table
- FR-7.1 Default command prints a human-readable table: resource address, monthly cost, and (for `--period hour`) hourly cost, with an indented component sub-list under each resource when `--show-components` (or `-v`) is set.
- FR-7.2 Footer shows project total (monthly/hourly) and a "N resources / M components not estimated" line.
- FR-7.3 A "Not estimated" section lists each unpriced in-plan resource or component and the reason (unsupported type, missing attribute, unknown-after-apply, no usage data, pricing error).
- FR-7.4 Respect `NO_COLOR` and non-TTY output (plain text, no ANSI).
- FR-7.5 `--period hour|month` controls which cost column is emphasised (default month).

### FR-8 — Output: JSON
- FR-8.1 `--format json` emits a stable, versioned JSON document (`schema_version`, `currency`, `resources[]`, `total_monthly`, `total_hourly`, `not_estimated[]`, `summary`, `metadata` incl. tool version, timestamp, plan format version).
- FR-8.2 JSON is the contract for CI consumption; field additions are backward compatible, removals/renames bump `schema_version`.
- FR-8.3 `--out <file>` writes output to a file instead of stdout, for any format.

### FR-9 — Output: diff view
- FR-9.1 `price-checker diff` compares two cost states and shows added / removed / changed resources with prior cost, new cost, and delta (monthly and hourly), plus a total delta.
- FR-9.2 Inputs: `--path` (new) and `--compare-to` (baseline) each accepting a plan JSON, a Terraform directory, or a previously produced price-checker JSON output.
- FR-9.3 When a single plan JSON contains both prior and planned state, `diff` can derive the change from that one file (`--from-plan`).
- FR-9.4 Diff is available in table and JSON formats.

### FR-10 — Output: HTML report
- FR-10.1 `--format html` (or `price-checker report --format html`) produces a self-contained HTML file (inline CSS/JS, no external requests) with: summary totals, per-module breakdown, per-resource component tables, not-estimated section, and — when run as a diff — the change view.
- FR-10.2 The report embeds the generating metadata (tool version, timestamp, region(s), whether a usage file was used).

### FR-11 — CI integration
- FR-11.1 Exit codes: `0` success; `1` runtime error (bad input, unreadable plan); `2` `--strict` and at least one component not estimated; `3` cost threshold breached (see FR-11.2). Documented and stable.
- FR-11.2 `--threshold-monthly <amount>` fails the run (exit 3) if the estimated project monthly total exceeds the amount; `--threshold-diff-monthly <amount>` fails if a diff's monthly delta exceeds it.
- FR-11.3 `--format github-comment` emits Markdown suitable for posting as a PR comment (summary table + collapsible detail + diff when applicable).
- FR-11.4 A documented recipe (not necessarily code in v1) for wiring `github-comment` output into a GitHub Actions workflow.

### FR-12 — CLI surface
- FR-12.1 Subcommands: `breakdown` (default), `diff`, `report`, `usage generate`, `version`, `help`.
- FR-12.2 Global flags: `--path`, `--compare-to`, `--format {table,json,html,github-comment}`, `--out`, `--period {hour,month}`, `--usage-file`, `--aws-region`, `--cache-dir`, `--cache-ttl`, `--no-cache`, `--refresh-cache`, `--strict`, `--threshold-monthly`, `--threshold-diff-monthly`, `--show-components/-v`, `--log-level`, `--no-color`.
- FR-12.3 Config file support (`price-checker.yml` in the working directory or `--config`) providing defaults for any global flag; precedence: flag > env var > config file > built-in default.
- FR-12.4 `--log-level` controls diagnostic logging on stderr; normal output stays on stdout.

### FR-13 — Coverage / not-estimated reporting
- FR-13.1 Every resource present in the plan that received no cost estimate is reported in the not-estimated output (table, JSON, HTML) with a machine-readable reason code and human message.
- FR-13.2 Reason codes: `UNSUPPORTED_TYPE`, `MISSING_ATTRIBUTE`, `UNKNOWN_AFTER_APPLY`, `NO_USAGE_DATA`, `PRICING_API_ERROR`, `UNSUPPORTED_CONFIGURATION`.
- FR-13.3 Summary counts of estimated vs not-estimated resources and components appear in all formats.

---

## Non-Functional Requirements

### NFR-1 — Performance
- NFR-1.1 A plan with 200 resources completes in ≤ 10 s with a warm cache, ≤ 60 s cold (network-bound), on a typical laptop.
- NFR-1.2 Price List queries for a run are deduplicated and issued concurrently with a bounded worker pool (default 8), respecting API throttling.

### NFR-2 — Reliability / correctness
- NFR-2.1 Monetary calculations use decimal arithmetic; results are stable and reproducible for identical inputs + cache.
- NFR-2.2 Partial failure is non-fatal by default: the tool still returns costs for everything it could price.
- NFR-2.3 Property-based tests (Partial mode) cover: usage-file YAML round-trip (PBT-02), JSON output round-trip (PBT-02), cost-aggregation invariants — total equals sum of components, non-negative, currency-consistent (PBT-03), and price-dimension parsing (PBT-02/03). Domain generators for plan resources and usage entries (PBT-07). Seeded, shrinking-enabled, in CI (PBT-08). Framework: `pgregory.net/rapid` (PBT-09).

### NFR-3 — Security / privacy
- NFR-3.1 No credentials, plan contents, or resource identifiers are transmitted anywhere except AWS Price List API calls (which carry only service/filter metadata, never plan data).
- NFR-3.2 Plan JSON may contain secrets in resource attributes; the tool never writes raw plan attribute values into cache, logs (above debug), or reports beyond what is needed to identify a resource and its cost drivers.
- NFR-3.3 The HTML report and github-comment output must not embed attribute values that are not cost-relevant.
- NFR-3.4 Cache files are created with user-only permissions (0600 / 0700).

### NFR-4 — Usability
- NFR-4.1 First-run with no AWS credentials produces an actionable error naming the credential-chain options and `--aws-region`.
- NFR-4.2 Help text documents every flag, exit code, and the usage-file schema.
- NFR-4.3 Table output fits 100 columns and degrades gracefully in narrow terminals.

### NFR-5 — Maintainability / extensibility
- NFR-5.1 Adding a new AWS resource type to the catalog requires implementing one well-defined interface (resource → cost components + Price List filters) and registering it; no changes to the engine or renderers.
- NFR-5.2 Price List service-code/filter mappings are declarative and unit-tested against recorded API fixtures.
- NFR-5.3 Output renderers consume only the engine's structured breakdown type, not Terraform or Price List types.
- NFR-5.4 Target Go version: latest stable (1.23+). Standard `go test`, `go vet`, `gofmt`; linting via `golangci-lint`.

### NFR-6 — Portability
- NFR-6.1 Single static binary for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64.
- NFR-6.2 No runtime dependencies except (optionally) the `terraform` binary for directory-input mode.

### NFR-7 — Observability
- NFR-7.1 `--log-level debug` reports each Price List query, cache hit/miss, and per-resource pricing decision on stderr.
- NFR-7.2 `--format json` metadata records cache hit ratio and query count for a run.

---

## Out of Scope for v1

- Non-AWS providers (GCP, Azure, Kubernetes, SaaS).
- Reserved Instances, Savings Plans, Spot pricing, private/EDP discounts, credits, free-tier modelling.
- Configurable / non-USD currency and FX conversion.
- Raw HCL parsing without a Terraform plan.
- Terraform state-file (`.tfstate`) input.
- A hosted service, dashboard, or database of past runs.
- Automatic posting of PR comments (the tool emits the Markdown; posting is the CI pipeline's job).

---

## Key Requirements Summary

- CLI tool in Go that reads a Terraform plan JSON (file, stdin, or via `terraform show -json` on a directory) and estimates monthly + hourly AWS cost in USD.
- Live pricing from the AWS Price List API via the SDK credential chain, with a TTL'd on-disk cache.
- Fixed v1 catalog of ~16 AWS resource groups; a pluggable interface for adding more; everything uncatalogued or under-specified is explicitly reported as "not estimated" with a reason code.
- Usage-based components priced only when supplied via an optional Infracost-style usage YAML; a generator subcommand scaffolds it.
- Four output formats: table, stable versioned JSON, self-contained HTML, GitHub PR-comment Markdown; plus a `diff` mode over two plans / prior outputs.
- CI-oriented: stable exit codes, cost thresholds that fail the build, `--strict` mode.
- Decimal money math; Partial property-based testing (round-trip, invariant, generator, shrinking, framework) with `rapid`.
- Security/resiliency extensions off; still observe basic secret-handling and least-permission cache file hygiene.
