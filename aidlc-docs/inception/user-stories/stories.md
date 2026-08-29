# User Stories — Price Checker

**Organization**: User Journey-Based epics. **Granularity**: Medium. **IDs**: `EPIC-<area>` / `US-<area>-<n>`.
**Acceptance criteria**: bulleted, verifiable. **Scope tag**: `v1` (in the first release) or `later` (deferred).
**Traceability**: each story links requirement IDs from `requirements.md` and maps personas from `personas.md`.
Cross-cutting NFRs (NFR-2.3 PBT, NFR-5 extensibility, NFR-6 portability) are attached as acceptance criteria on their most-related story (clarification 1 = A).

Personas: **P1** Local Developer (Dana) · **P2** Platform/DevOps Engineer (Sam) · **P3** CI Pipeline · **P4** PR Reviewer (Priya) · **P5** FinOps/Cost Owner (Alex).

---

## EPIC-LOCAL-ESTIMATE — Estimate the cost of a plan locally

Goal: a developer gets a trustworthy cost breakdown of a Terraform change on their machine with minimal setup.

### US-LOCAL-ESTIMATE-1 — Provide a plan as a file or on stdin
**Tag**: v1 · **Requirements**: FR-1.1, FR-1.2, FR-1.4, FR-1.5, FR-6.5 · **Personas**: P1, P2, P3

As a developer, I want to give the tool a Terraform plan JSON by path or by piping it in, so that I can price whichever plan I already have.

**Acceptance criteria**
- `price-checker --path plan.json` reads and prices the file.
- With no `--path`, or `--path -`, the tool reads plan JSON from stdin.
- Plan JSON from Terraform 1.x (`format_version` 0.1 and 1.x) is accepted; an unknown/older format exits with code 1 and a message naming the detected version.
- Malformed JSON exits with code 1 and a message pointing at the parse location.
- Planned resources are extracted with type, name, address, module path, and resolved attribute values; resources whose change action is delete-only contribute zero cost.
- Output ordering is deterministic (by resource address).

### US-LOCAL-ESTIMATE-2 — Price a Terraform directory directly
**Tag**: v1 · **Requirements**: FR-1.3 · **Personas**: P1, P2

As a developer, I want to point the tool at my Terraform directory, so that I don't have to generate the plan JSON myself.

**Acceptance criteria**
- `price-checker --path ./infra` where `./infra` is a directory runs `terraform show -json` (generating a plan file as needed) and prices the result.
- If the `terraform` binary is not on PATH, the tool exits with code 1 and a message saying directory mode needs Terraform installed, and that a plan JSON file/stdin works without it.
- `terraform` errors are surfaced verbatim on stderr with a non-zero exit.
- The working directory is left clean (any temp plan file is removed).

### US-LOCAL-ESTIMATE-3 — Read a cost breakdown table
**Tag**: v1 · **Requirements**: FR-7.1, FR-7.2, FR-7.4, FR-6.1, FR-6.2, FR-6.3 · **Personas**: P1, P2

As a developer, I want a readable table of costs per resource with a total, so that I can see the impact at a glance.

**Acceptance criteria**
- Default output is a table: resource address and monthly cost, one row per priced resource.
- A footer shows the project monthly total and a `N resources / M components not estimated` line.
- Totals are grouped by module and then a grand total.
- The grand total includes only estimated components; the not-estimated count is stated separately.
- Output respects `NO_COLOR` and non-TTY (plain text, no ANSI); data goes to stdout, diagnostics to stderr.
- Table fits within 100 columns and degrades gracefully in narrow terminals.

### US-LOCAL-ESTIMATE-4 — Expand per-resource cost components and choose the period
**Tag**: v1 · **Requirements**: FR-7.1, FR-7.5, FR-6.1, FR-6.4, FR-12.2 · **Personas**: P1, P5

As a developer, I want to see the individual cost components of a resource and switch between hourly and monthly, so that I understand what drives the cost.

**Acceptance criteria**
- `--show-components` / `-v` prints an indented component list under each resource: component name, unit, unit price, monthly quantity, monthly cost (and hourly cost where the unit is hourly).
- `--period month` (default) emphasises monthly figures; `--period hour` emphasises hourly.
- Monthly figures use a 730-hour month.
- All monetary values use decimal arithmetic; repeated runs on the same input + cache produce identical figures.
- Currency is USD and is labelled as such in the output.

---

## EPIC-PRICING — Get real prices for AWS resources

Goal: costs reflect current AWS on-demand pricing, and anything that can't be priced is called out explicitly.

### US-PRICING-1 — Live on-demand prices from the AWS Price List API
**Tag**: v1 · **Requirements**: FR-3.1, FR-3.4 · **Personas**: P1, P2, P5

As a user, I want prices pulled from AWS directly, so that estimates track current published rates without me updating a price sheet.

**Acceptance criteria**
- Prices are retrieved via the AWS Price List Query API (`GetProducts`/`GetAttributeValues`) using AWS SDK for Go v2.
- Each catalogued resource + attribute set maps to a service code and filter set, selecting the on-demand price dimension.
- Price List service-code/filter mappings are declarative and unit-tested against recorded API fixtures (NFR-5.2).
- Reserved Instances, Savings Plans, Spot, private/EDP discounts, credits and free tier are explicitly not applied (documented in `--help` and report metadata).

### US-PRICING-2 — Use my AWS credentials with a clear failure message
**Tag**: v1 · **Requirements**: FR-3.2, NFR-4.1, NFR-3.1 · **Personas**: P1, P2, P3

As a user, I want the tool to use my normal AWS credentials, so that I don't configure anything extra.

**Acceptance criteria**
- Credentials resolve through the standard SDK chain: environment variables, shared config/credentials files, named profiles, SSO, and container/instance roles.
- On missing/invalid credentials the tool exits code 1 with an actionable message listing the chain options and `--aws-region`.
- Only service/filter metadata is sent to AWS; no plan contents, resource names, or attribute values are transmitted (NFR-3.1).

### US-PRICING-3 — Price each resource in its own region
**Tag**: v1 · **Requirements**: FR-3.3 · **Personas**: P1, P2

As a user with multi-region infrastructure, I want each resource priced in its region, so that the estimate is accurate.

**Acceptance criteria**
- Region per resource is derived from the plan's provider configuration (including aliased providers).
- `--aws-region <r>` overrides the region for all resources.
- If a resource's region cannot be determined and no override is given, that resource is reported not-estimated with reason `MISSING_ATTRIBUTE`.
- Price List API calls target a valid Price List endpoint regardless of the priced region.

### US-PRICING-4 — Cover the core AWS cost drivers
**Tag**: v1 · **Requirements**: FR-2, FR-2.1, FR-2.3 · **Personas**: P1, P2, P5

As a user, I want the common AWS resources in my plans to be priced, so that the total is meaningful.

**Acceptance criteria**
- The v1 catalog prices: `aws_instance`, `aws_ebs_volume`, `aws_ebs_snapshot`/`aws_ebs_snapshot_copy`, `aws_eip`, `aws_db_instance`, `aws_rds_cluster`/`aws_rds_cluster_instance`, `aws_s3_bucket`, `aws_lambda_function`, `aws_lb`/`aws_alb`/`aws_elb`, `aws_nat_gateway`, `aws_eks_cluster`, `aws_eks_node_group`, `aws_elasticache_cluster`/`aws_elasticache_replication_group`, `aws_dynamodb_table`, and `aws_cloudwatch_*` (log group, metric alarm, dashboard).
- For each, the cost components listed in requirements FR-2 are computed (fixed components always; usage-based components only with usage data — see EPIC-USAGE).
- `count` and `for_each` expansions are priced per instance.
- Instance/volume/node counts and types read from resolved plan attributes.

### US-PRICING-5 — See exactly what was not estimated and why
**Tag**: v1 · **Requirements**: FR-2.2, FR-7.3, FR-13.1, FR-13.2, FR-13.3 · **Personas**: P1, P2, P4, P5

As a user, I want unpriced resources and components listed with a reason, so that I know the total's blind spots.

**Acceptance criteria**
- Every resource in the plan that received no estimate appears in a "Not estimated" section (table, JSON, HTML) with a machine-readable reason code and a human message.
- Reason codes: `UNSUPPORTED_TYPE`, `MISSING_ATTRIBUTE`, `UNKNOWN_AFTER_APPLY`, `NO_USAGE_DATA`, `PRICING_API_ERROR`, `UNSUPPORTED_CONFIGURATION`.
- Attributes unknown at plan time (`known after apply`) fall back to a usage value if present, otherwise mark that component `UNKNOWN_AFTER_APPLY`.
- Summary counts of estimated vs not-estimated resources and components appear in all formats.
- A partially-priced resource shows its known components and lists its missing components as not-estimated.

### US-PRICING-6 — Keep working when the pricing API is slow or throttled
**Tag**: v1 · **Requirements**: FR-3.5, NFR-1.2, NFR-2.2 · **Personas**: P1, P3

As a user on a flaky network, I want the tool to degrade gracefully, so that one failed lookup doesn't kill the whole run.

**Acceptance criteria**
- Price List queries for a run are deduplicated and issued concurrently with a bounded worker pool (default 8, configurable), backing off on throttling responses.
- A query that still fails after retries marks only its affected resources not-estimated with reason `PRICING_API_ERROR`; the rest of the run completes.
- Default exit code stays 0 on partial failure (unless `--strict` — see US-CI-GATE-2).
- A 200-resource plan completes in ≤ 10 s warm-cache and ≤ 60 s cold-cache on a typical laptop (NFR-1.1).

### US-PRICING-7 — Add a new resource type without changing the engine
**Tag**: v1 · **Requirements**: NFR-5.1, NFR-5.3 · **Personas**: P2

As a platform engineer, I want to extend coverage by implementing one interface, so that adopting a new AWS service doesn't mean forking the tool.

**Acceptance criteria**
- A new resource type is added by implementing a single documented interface (resource attributes → cost components + Price List filters) and registering it; no changes to the cost engine or output renderers.
- Output renderers consume only the engine's structured breakdown type, not Terraform or Price List types (NFR-5.3).
- The catalog registration point and the interface are documented with a worked example in the repo.
- A contract test verifies every registered pricer produces components that the engine and all renderers can consume.

---

## EPIC-CACHE — Fast repeat runs

Goal: pricing data is reused between runs so iteration is fast, with explicit control over freshness.

### US-CACHE-1 — Cache pricing responses on disk
**Tag**: v1 · **Requirements**: FR-4.1, NFR-1.1, NFR-3.4 · **Personas**: P1, P2

As a developer running the tool repeatedly, I want pricing data cached locally, so that warm runs are fast and offline-tolerant for unchanged lookups.

**Acceptance criteria**
- Price List responses are cached under `~/.price-checker/cache` by default, overridable via `--cache-dir` or `PRICE_CHECKER_CACHE_DIR`.
- Cache files and directories are created with user-only permissions (0600 / 0700).
- No raw plan attribute values are written to the cache — only Price List request keys and responses.
- A warm run for an unchanged plan issues zero Price List queries.

### US-CACHE-2 — Control cache freshness
**Tag**: v1 · **Requirements**: FR-4.2, FR-4.3 · **Personas**: P1, P2, P3

As a user, I want to control how stale cached prices can be, so that I balance speed against accuracy.

**Acceptance criteria**
- Each cache entry stores a timestamp; entries older than the TTL (default 7 days, set via `--cache-ttl`) are refetched.
- `--no-cache` bypasses both cache read and write.
- `--refresh-cache` ignores existing entries on read but writes fresh results.
- The JSON metadata reports cache hit ratio and query count for the run (NFR-7.2).

### US-CACHE-3 — Safe under concurrent and corrupted state
**Tag**: v1 · **Requirements**: FR-4.4 · **Personas**: P2, P3

As a CI maintainer running parallel jobs, I want the cache to tolerate concurrency and corruption, so that runs don't fail or poison each other.

**Acceptance criteria**
- Cache writes are atomic (write-temp-then-rename).
- Two concurrent runs sharing a cache dir both succeed and neither observes a partially written entry.
- A corrupt or unreadable cache entry is treated as a miss and refetched, not a fatal error.

---

## EPIC-USAGE — Model usage-based costs

Goal: usage-driven costs (S3 storage, Lambda invocations, NAT data processed, etc.) are estimated when the user supplies assumptions.

### US-USAGE-1 — Supply monthly usage assumptions
**Tag**: v1 · **Requirements**: FR-5.1, FR-5.2, FR-5.3 · **Personas**: P2, P5

As a cost owner, I want to provide monthly usage numbers in a file, so that usage-based components are included in the estimate.

**Acceptance criteria**
- `--usage-file <path>` accepts a YAML file modelled on Infracost's `infracost-usage.yml` (`version`, `resource_usage` keyed by resource address, per-component monthly quantities).
- Documented usage keys map to the usage-based components in FR-2 (e.g. `monthly_data_processed_gb`, `monthly_requests`, `request_duration_ms`, `storage_gb`).
- Components with a supplied usage value are priced and appear as normal line items.
- Components with no supplied value remain not-estimated with reason `NO_USAGE_DATA` (no built-in default assumptions in v1).

### US-USAGE-2 — Get warnings, not failures, for a stale usage file
**Tag**: v1 · **Requirements**: FR-5.4 · **Personas**: P2, P5

As a platform engineer maintaining a shared usage file, I want non-fatal validation feedback, so that infra changes don't break every run.

**Acceptance criteria**
- Unknown resource addresses in the usage file produce a warning on stderr and are ignored.
- Unknown usage keys produce a warning naming the key and resource and are ignored.
- Malformed YAML is a fatal error (exit 1) with the parse location.
- Warnings do not change the exit code.

### US-USAGE-3 — Scaffold a usage file from a plan
**Tag**: v1 · **Requirements**: FR-5.5 · **Personas**: P1, P2, P5

As a user, I want a starter usage file generated from my plan, so that I know which assumptions the estimate needs.

**Acceptance criteria**
- `price-checker usage generate --path <plan>` emits a YAML skeleton listing every usage-based component found in the plan, keyed by resource address, with empty values and an inline comment per key describing the unit.
- `--out <file>` writes it to a file; otherwise stdout.
- Resources with no usage-based components are omitted from the skeleton.
- Running with the generated file (values still empty) produces the same result as running with no usage file.

---

## EPIC-MACHINE-OUTPUT — Machine-readable output

Goal: automation can consume a stable, versioned representation of the estimate.

### US-MACHINE-OUTPUT-1 — Stable versioned JSON
**Tag**: v1 · **Requirements**: FR-8.1, FR-8.2, FR-13.3, NFR-2.1, NFR-2.3 · **Personas**: P2, P3, P5

As an automation author, I want a documented JSON schema, so that I can build on the output without it breaking under me.

**Acceptance criteria**
- `--format json` emits `schema_version`, `currency`, `resources[]` (address, type, module, components, monthly + hourly cost), `total_monthly`, `total_hourly`, `not_estimated[]` (address, reason_code, message), `summary` (counts), and `metadata` (tool version, timestamp, plan format version, region(s), usage-file used, cache hit ratio, query count).
- Field additions are backward compatible; any removal or rename bumps `schema_version`.
- JSON output is deterministic for identical input + cache (stable key order, stable array order).
- Property-based tests (NFR-2.3): JSON output round-trips (marshal → unmarshal → equal); usage-file YAML round-trips; cost aggregation invariants hold (resource total equals sum of its components; totals are non-negative; a single currency throughout). Tests use domain generators for plan resources and usage entries, run with a logged seed, shrinking enabled, in CI. Framework: `pgregory.net/rapid` (NFR-2.3 / PBT-09).

### US-MACHINE-OUTPUT-2 — Write output to a file
**Tag**: v1 · **Requirements**: FR-8.3, FR-12.4 · **Personas**: P2, P3

As a pipeline author, I want to send output straight to a file, so that I can attach it as an artifact.

**Acceptance criteria**
- `--out <file>` writes the rendered output (any format) to the file; stdout stays clean for logs/redirection.
- Parent directory is created if missing; an unwritable path exits code 1 with a clear message.
- Diagnostic logging always goes to stderr regardless of `--out`.

---

## EPIC-REPORT — Shareable HTML report

### US-REPORT-1 — Self-contained HTML report
**Tag**: v1 · **Requirements**: FR-10.1, FR-10.2, NFR-3.3 · **Personas**: P4, P5

As a reviewer or cost owner, I want an HTML report I can open and share, so that I can inspect a breakdown without the CLI.

**Acceptance criteria**
- `--format html` (or `price-checker report --format html`) produces a single HTML file with inline CSS/JS and **no external network requests**.
- Sections: summary totals, per-module breakdown, per-resource component tables, not-estimated section, and — when run as a diff — the change view.
- The report embeds metadata: tool version, timestamp, region(s), whether a usage file was used.
- The report does not embed resource attribute values that are not cost-relevant (NFR-3.3).
- Opens correctly from `file://` in current Chrome, Firefox, and Safari.

---

## EPIC-DIFF — Cost change of a proposed change

### US-DIFF-1 — Compare two cost states
**Tag**: v1 · **Requirements**: FR-9.1, FR-9.2, FR-9.3 · **Personas**: P1, P2, P3, P4

As a reviewer, I want to see the cost delta a change introduces, so that I can judge whether it's acceptable.

**Acceptance criteria**
- `price-checker diff --path <new> --compare-to <baseline>` shows added / removed / changed resources with prior cost, new cost, and delta (monthly and hourly), plus a total delta.
- `--path` and `--compare-to` each accept a plan JSON, a Terraform directory, or a previously produced price-checker JSON output.
- `price-checker diff --from-plan --path <plan>` derives the change from a single plan JSON that contains both prior and planned state.
- A resource that is not-estimated on either side is shown as not-estimated in the diff, not as a spurious delta.

### US-DIFF-2 — Diff in every output format
**Tag**: v1 · **Requirements**: FR-9.4 · **Personas**: P3, P4, P5

As a pipeline, I want the diff available as table, JSON, and HTML, so that both humans and automation can use it.

**Acceptance criteria**
- `diff` supports `--format table|json|html|github-comment`.
- Diff JSON has its own documented schema block (`schema_version`, `changes[]` with `prior_monthly`/`new_monthly`/`delta_monthly`, `total_delta_monthly`) and is deterministic.
- Table diff uses sign-prefixed deltas and is `NO_COLOR`-safe.

---

## EPIC-CI-GATE — Fail the build on cost

### US-CI-GATE-1 — Predictable exit codes
**Tag**: v1 · **Requirements**: FR-11.1, NFR-7.1 · **Personas**: P2, P3

As a pipeline author, I want distinct exit codes, so that I can tell "the tool broke" from "the cost is too high".

**Acceptance criteria**
- Exit codes: `0` success; `1` runtime error (bad input, unreadable plan, credentials); `2` `--strict` with at least one not-estimated component; `3` a cost threshold was breached.
- Exit codes are documented in `--help` and the README and do not change between patch releases.
- `--log-level` controls stderr diagnostics; it never changes stdout data or the exit code.

### US-CI-GATE-2 — Threshold gates and strict mode
**Tag**: v1 · **Requirements**: FR-11.2, FR-7.3, FR-13.1 · **Personas**: P2, P5

As a cost owner, I want the build to fail when a change costs too much or can't be fully priced, so that surprises are caught in review.

**Acceptance criteria**
- `--threshold-monthly <amount>` exits code 3 if the estimated project monthly total exceeds the amount.
- `--threshold-diff-monthly <amount>` exits code 3 if a diff's monthly delta exceeds the amount.
- `--strict` exits code 2 if any in-plan resource or component could not be estimated, after printing the not-estimated section.
- Threshold breach and strict failure still print the full normal output before exiting.
- Precedence when multiple gates trip: runtime error (1) > strict (2) > threshold (3).

---

## EPIC-PR-COMMENT — Cost summary on the pull request

### US-PR-COMMENT-1 — Emit a PR comment body
**Tag**: v1 · **Requirements**: FR-11.3 · **Personas**: P2, P3

As a pipeline author, I want a Markdown comment body, so that my workflow can post it to the PR.

**Acceptance criteria**
- `--format github-comment` emits Markdown: a headline total (or headline delta for a diff), a compact table, and a `<details>` block with the full breakdown.
- Output is pure Markdown to stdout (or `--out`), with no posting performed by the tool.
- The comment includes the not-estimated count and the tool version/timestamp.
- Deterministic output so a pipeline can update-in-place instead of re-posting.

### US-PR-COMMENT-2 — Understand a change from the comment alone
**Tag**: v1 · **Requirements**: FR-11.3, FR-9.1 · **Personas**: P4

As a reviewer, I want the comment to lead with what matters, so that I can approve or push back quickly.

**Acceptance criteria**
- First line states the monthly delta (e.g. `Estimated monthly cost: +$142.50 (was $1,020.00 → $1,162.50)`).
- Changed resources are shown in a table sorted by absolute delta, largest first.
- A visible `N resources not estimated` note appears above the fold when the count is non-zero.
- Full detail is collapsed by default.

### US-PR-COMMENT-3 — CI wiring recipe
**Tag**: later · **Requirements**: FR-11.4 · **Personas**: P2

As a platform engineer, I want a documented GitHub Actions recipe, so that I can adopt cost checks quickly.

**Acceptance criteria**
- The README documents a working GitHub Actions job: assume role, generate base and PR plans, run `diff --format github-comment`, post/update the comment, and gate on exit code 3.
- The recipe notes the minimum IAM permissions (Price List read-only).

---

## EPIC-CONFIG-INSTALL — Install, configure, run consistently

### US-CONFIG-INSTALL-1 — Team-wide defaults via a config file
**Tag**: v1 · **Requirements**: FR-12.3 · **Personas**: P2, P5

As a platform engineer, I want a committed config file, so that everyone and CI get identical behaviour.

**Acceptance criteria**
- `price-checker.yml` in the working directory (or `--config <path>`) can set defaults for any global flag.
- Precedence is flag > environment variable > config file > built-in default, and this is documented.
- An unknown key in the config file is a warning, not a fatal error.
- `--log-level debug` prints the effective resolved configuration and where each value came from.

### US-CONFIG-INSTALL-2 — Discoverable CLI with clean output streams
**Tag**: v1 · **Requirements**: FR-12.1, FR-12.2, FR-12.4, NFR-4.2, NFR-4.3, NFR-7.1 · **Personas**: P1, P2, P3

As a user, I want a self-documenting CLI, so that I can use it without external docs.

**Acceptance criteria**
- Subcommands: `breakdown` (default), `diff`, `report`, `usage generate`, `version`, `help`.
- `--help` documents every global flag, every exit code, and the usage-file schema (or links to it in-repo).
- `version` / `--version` prints tool version, git commit, build date, and Go version.
- Normal output is on stdout; all logs and diagnostics on stderr; `--log-level` controls verbosity.
- `--log-level debug` reports each Price List query and each cache hit/miss and per-resource pricing decision (NFR-7.1).

### US-CONFIG-INSTALL-3 — Install a single binary anywhere
**Tag**: v1 · **Requirements**: NFR-6.1, NFR-6.2 · **Personas**: P1, P2, P3

As a user, I want one self-contained binary, so that installation is a download with no runtime dependencies.

**Acceptance criteria**
- Release builds are published as a single static binary for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64.
- The binary runs with no runtime dependencies; the only optional external dependency is the `terraform` binary for directory-input mode (US-LOCAL-ESTIMATE-2).
- `price-checker version` works offline.
- Built with the latest stable Go (1.23+); `go vet`, `gofmt`, and `golangci-lint` are clean in CI (NFR-5.4).

---

## Persona → Story Traceability

| Story | P1 | P2 | P3 | P4 | P5 |
|---|:--:|:--:|:--:|:--:|:--:|
| US-LOCAL-ESTIMATE-1 | ✅ | ✅ | ✅ |  |  |
| US-LOCAL-ESTIMATE-2 | ✅ | ✅ |  |  |  |
| US-LOCAL-ESTIMATE-3 | ✅ | ✅ |  |  |  |
| US-LOCAL-ESTIMATE-4 | ✅ |  |  |  | ✅ |
| US-PRICING-1 | ✅ | ✅ |  |  | ✅ |
| US-PRICING-2 | ✅ | ✅ | ✅ |  |  |
| US-PRICING-3 | ✅ | ✅ |  |  |  |
| US-PRICING-4 | ✅ | ✅ |  |  | ✅ |
| US-PRICING-5 | ✅ | ✅ |  | ✅ | ✅ |
| US-PRICING-6 | ✅ |  | ✅ |  |  |
| US-PRICING-7 |  | ✅ |  |  |  |
| US-CACHE-1 | ✅ | ✅ |  |  |  |
| US-CACHE-2 | ✅ | ✅ | ✅ |  |  |
| US-CACHE-3 |  | ✅ | ✅ |  |  |
| US-USAGE-1 |  | ✅ |  |  | ✅ |
| US-USAGE-2 |  | ✅ |  |  | ✅ |
| US-USAGE-3 | ✅ | ✅ |  |  | ✅ |
| US-MACHINE-OUTPUT-1 |  | ✅ | ✅ |  | ✅ |
| US-MACHINE-OUTPUT-2 |  | ✅ | ✅ |  |  |
| US-REPORT-1 |  |  |  | ✅ | ✅ |
| US-DIFF-1 | ✅ | ✅ | ✅ | ✅ |  |
| US-DIFF-2 |  |  | ✅ | ✅ | ✅ |
| US-CI-GATE-1 |  | ✅ | ✅ |  |  |
| US-CI-GATE-2 |  | ✅ |  |  | ✅ |
| US-PR-COMMENT-1 |  | ✅ | ✅ |  |  |
| US-PR-COMMENT-2 |  |  |  | ✅ |  |
| US-PR-COMMENT-3 |  | ✅ |  |  |  |
| US-CONFIG-INSTALL-1 |  | ✅ |  |  | ✅ |
| US-CONFIG-INSTALL-2 | ✅ | ✅ | ✅ |  |  |
| US-CONFIG-INSTALL-3 | ✅ | ✅ | ✅ |  |  |

## Requirements → Story Coverage

| Requirement | Stories |
|---|---|
| FR-1 (plan ingestion) | US-LOCAL-ESTIMATE-1, US-LOCAL-ESTIMATE-2 |
| FR-2 (pricing catalog) | US-PRICING-4, US-PRICING-1, US-USAGE-1 |
| FR-3 (pricing retrieval) | US-PRICING-1, US-PRICING-2, US-PRICING-3, US-PRICING-6 |
| FR-4 (cache) | US-CACHE-1, US-CACHE-2, US-CACHE-3 |
| FR-5 (usage file) | US-USAGE-1, US-USAGE-2, US-USAGE-3 |
| FR-6 (cost engine) | US-LOCAL-ESTIMATE-1, US-LOCAL-ESTIMATE-3, US-LOCAL-ESTIMATE-4 |
| FR-7 (table output) | US-LOCAL-ESTIMATE-3, US-LOCAL-ESTIMATE-4, US-PRICING-5 |
| FR-8 (JSON output) | US-MACHINE-OUTPUT-1, US-MACHINE-OUTPUT-2 |
| FR-9 (diff) | US-DIFF-1, US-DIFF-2, US-PR-COMMENT-2 |
| FR-10 (HTML report) | US-REPORT-1 |
| FR-11 (CI integration) | US-CI-GATE-1, US-CI-GATE-2, US-PR-COMMENT-1, US-PR-COMMENT-3 |
| FR-12 (CLI surface) | US-CONFIG-INSTALL-1, US-CONFIG-INSTALL-2, US-LOCAL-ESTIMATE-4, US-MACHINE-OUTPUT-2 |
| FR-13 (coverage reporting) | US-PRICING-5, US-CI-GATE-2, US-MACHINE-OUTPUT-1 |
| NFR-1 (performance) | US-PRICING-6, US-CACHE-1 |
| NFR-2 (reliability/correctness) | US-LOCAL-ESTIMATE-4, US-PRICING-6, US-MACHINE-OUTPUT-1 |
| NFR-2.3 (property-based testing) | US-MACHINE-OUTPUT-1 |
| NFR-3 (security/privacy) | US-PRICING-2, US-CACHE-1, US-REPORT-1 |
| NFR-4 (usability) | US-PRICING-2, US-CONFIG-INSTALL-2, US-LOCAL-ESTIMATE-3 |
| NFR-5 (maintainability/extensibility) | US-PRICING-7, US-PRICING-1, US-CONFIG-INSTALL-3 |
| NFR-6 (portability) | US-CONFIG-INSTALL-3 |
| NFR-7 (observability) | US-CI-GATE-1, US-CACHE-2, US-CONFIG-INSTALL-2 |

## INVEST Check

All 30 stories reviewed against INVEST:
- **Independent**: stories share the engine but each delivers a slice usable on its own; ordering dependencies (e.g. diff builds on breakdown) are noted, not blocking.
- **Negotiable**: acceptance criteria state outcomes, not implementation.
- **Valuable**: each story names the persona value in its narrative.
- **Estimable**: scope is bounded to one journey step or output format.
- **Small**: medium granularity; each fits within a single unit of work.
- **Testable**: every criterion is observable via CLI output, exit code, file content, or a test fixture.

## Deferred to "later" (not in v1)

- US-PR-COMMENT-3 — GitHub Actions wiring recipe (docs; the `github-comment` output itself is v1).
