# price-checker

Estimate the AWS cost of a Terraform plan — an Infracost-style CLI. Reads a
Terraform plan JSON (file, stdin, or a directory it runs `terraform show -json`
against), prices the AWS resources against the **live AWS Price List API**, and
prints a monthly + hourly USD estimate. Anything it cannot price is listed
explicitly with a reason.

> Status: v1 in development via the AI-DLC workflow (see `aidlc-docs/`). Build
> requires Go 1.23+; run `go mod tidy` after cloning.

## Install

Download a static binary from the releases page, or:

```
go install github.com/osifotos/price-checker/cmd/price-checker@latest
```

No runtime dependencies. `terraform` is only needed for directory input mode.

## Usage

```
price-checker --path plan.json                 # table
terraform show -json | price-checker            # stdin
price-checker --path ./infra                    # runs terraform for you
price-checker --path plan.json --format json    # machine-readable
price-checker report --path plan.json           # self-contained HTML
price-checker diff --path new.json --compare-to base.json
price-checker diff --from-plan --path plan.json
price-checker usage generate --path plan.json > infra-usage.yml
```

AWS credentials come from the standard SDK chain (environment, shared config,
`--profile`, SSO, instance role). The Price List API is USD-only; v1 shows USD.

### Live AWS verification

For real AWS Price List validation, opt in explicitly with `--live` or the
`PRICE_CHECKER_LIVE=1` environment variable:

```bash
price-checker live-verify --live --aws-region us-east-1
# or
PRICE_CHECKER_LIVE=1 price-checker live-verify
```

This performs a minimal live `pricing:GetProducts` probe against the configured
AWS region and fails clearly if credentials are missing or invalid. Normal runs
remain offline by default, so you do not need AWS credentials for standard plan
estimation.

### Key flags

| Flag | Meaning |
|---|---|
| `--path, -p` | plan JSON file, Terraform directory, or `-` for stdin |
| `--format, -f` | `table` (default), `json`, `html`, `github-comment` |
| `--out, -o` | write output to a file |
| `--period` | `month` (default) or `hour` |
| `--show-components, -v` | show per-resource cost components |
| `--usage-file` | usage assumptions YAML (see below) |
| `--aws-region` | override the region for every resource |
| `--live` / `PRICE_CHECKER_LIVE` | opt in to a real AWS Price List verification query using configured credentials |
| `--cache-dir` / `--cache-ttl` / `--no-cache` / `--refresh-cache` | on-disk price cache (default `~/.price-checker/cache`, 7-day TTL) |
| `--strict` | exit 2 if anything could not be estimated |
| `--threshold-monthly` / `--threshold-diff-monthly` | exit 3 if exceeded |
| `--concurrency` | parallel pricing requests (default 8) |
| `--log-level` | `error` \| `warn` (default) \| `info` \| `debug` |
| `--config` | config file (default `./price-checker.yml`) |

All flags can be set via `PRICE_CHECKER_*` env vars or the config file.
Precedence: **flag / env > config file > default**.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | success |
| `1` | runtime error (bad input, unreadable plan, credentials) |
| `2` | `--strict` and at least one component was not estimated |
| `3` | a `--threshold-*` limit was exceeded |

## Usage file

Usage-based costs (S3 storage, Lambda invocations, NAT data processed, …) are
only estimated when you supply monthly assumptions, in an Infracost-style YAML:

```yaml
version: "0.1"
resource_usage:
  aws_s3_bucket.assets:
    storage_gb: 512
    monthly_tier1_requests: 100000
  module.net.aws_nat_gateway.this:
    monthly_data_processed_gb: 250
```

`price-checker usage generate --path plan.json` scaffolds one listing every
usage-based component in your plan.

## JSON output

`--format json` emits a versioned document. The schema is committed at
`pkg/schema/breakdown.schema.json` / `diff.schema.json` / `usage.schema.json`
and validated in the test suite. Field additions are backward compatible;
`schema_version` bumps on any breaking change.

## GitHub Actions (cost check on PRs)

```yaml
# .github/workflows/cost.yml  (recipe — adjust to your setup)
name: cost
on: pull_request
permissions: { contents: read, pull-requests: write, id-token: write }
jobs:
  cost:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aws-actions/configure-aws-credentials@v4
        with: { role-to-assume: ${{ secrets.PRICING_READONLY_ROLE }}, aws-region: us-east-1 }
      - uses: hashicorp/setup-terraform@v3
      - name: base plan
        run: |
          git worktree add ../base ${{ github.event.pull_request.base.sha }}
          terraform -chdir=../base/infra init -input=false
          terraform -chdir=../base/infra plan -out base.tfplan
          terraform -chdir=../base/infra show -json base.tfplan > "$GITHUB_WORKSPACE/base.json"
      - name: pr plan
        run: |
          terraform -chdir=infra init -input=false
          terraform -chdir=infra plan -out pr.tfplan
          terraform -chdir=infra show -json pr.tfplan > pr.json
      - name: cost diff
        run: |
          price-checker diff --path pr.json --compare-to base.json \
            --format github-comment --out comment.md \
            --threshold-diff-monthly 100
      - uses: marocchino/sticky-pull-request-comment@v2
        with: { path: comment.md }
```

The minimum IAM permission is `pricing:GetProducts` (read-only).

## Minimum IAM policy

```json
{ "Version": "2012-10-17", "Statement": [
  { "Effect": "Allow", "Action": ["pricing:GetProducts", "pricing:GetAttributeValues"], "Resource": "*" }
]}
```

## Scope (v1)

AWS only. Priced: EC2, EBS (+snapshots, EIP/public IPv4), RDS + Aurora,
ElastiCache, DynamoDB (provisioned), S3, Lambda, ALB/NLB/CLB, NAT Gateway, EKS
(cluster + node groups), CloudWatch (alarms, dashboards), SQS. Not modelled:
Reserved Instances / Savings Plans / Spot, private pricing, free tier,
non-AWS providers, non-USD currency, Cognito, API Gateway (v1/v2), EventBridge.

Resource types that never carry a direct AWS charge of their own (route
tables, security groups, subnets, VPCs, IAM roles/policies, an ECS cluster,
etc.) are reported with reason `NOT_BILLABLE` rather than `UNSUPPORTED_TYPE` —
$0.00 is the correct, final answer for these.
