# Requirements Verification Questions — Price Checker (Terraform Cloud Cost Calculator)

Please answer each question by filling in the letter choice after the `[Answer]:` tag.
If none of the options match, choose the last option (Other) and describe your preference after the `[Answer]:` tag.
Let me know when you're done.

Already settled from earlier conversation (no need to re-answer):
- Input format: Terraform plan JSON (`terraform show -json`)
- Cloud provider scope: AWS core services
- Pricing data source: Live AWS Price List API
- Implementation language: Go
- Reference product: Infracost

---

## Question 1
Besides a plan JSON file, which other input modes should the first version accept?

A) Plan JSON only (a `--path` to a file, or piped via stdin)

B) Plan JSON file/stdin, plus the ability to run `terraform show -json` automatically against a given Terraform directory

C) Plan JSON file/stdin, plus raw HCL directory parsing (no terraform binary needed)

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 2
What output formats are required for v1?

A) Human-readable CLI table only

B) CLI table + JSON

C) CLI table + JSON + a diff/"cost change" view (compare two plans or before/after within one plan, like `infracost diff`)

D) CLI table + JSON + diff + HTML report

X) Other (please describe after [Answer]: tag below)

[Answer]: D

---

## Question 3
Which AWS services must be covered in the v1 "AWS core" scope? (Choose the set closest to your need.)

A) Compute + storage essentials: EC2 instances, EBS volumes, EBS snapshots, Elastic IPs

B) Option A plus: RDS instances + storage, S3 (storage + requests), Lambda, ELB/ALB/NLB, NAT Gateway

C) Option B plus: EKS, ElastiCache, CloudWatch, DynamoDB, data transfer

X) Other (please describe / list exact resource types after [Answer]: tag below)

[Answer]: All resources depending on what it scanned from the project folder

---

## Question 4
How should usage-driven costs (e.g. S3 GB stored, Lambda invocations, data transfer GB, NAT Gateway GB processed) be handled in v1?

A) Show only price rates + fixed monthly costs; mark usage-based line items as "usage-based, not estimated"

B) Support an optional usage file (like Infracost's `infracost-usage.yml`) where the user supplies monthly usage assumptions

C) Option B, plus built-in default usage assumptions when no usage file is provided

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 5
How should the tool obtain AWS credentials / region for calling the Price List API?

A) Standard AWS SDK credential chain (env vars, shared config, profiles) + region from plan/provider config, overridable by `--aws-region`

B) Option A, but also allow a no-credentials mode using the public bulk pricing JSON endpoint (no AWS account needed)

C) Require an explicit `--profile` / `--region` flag every run

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 6
The AWS Price List API is slow and rate-limited. What caching behavior do you want?

A) In-memory cache for a single run only

B) On-disk cache (e.g. `~/.price-checker/cache`) with a configurable TTL, plus `--no-cache` and `--refresh-cache` flags

C) No caching in v1

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 7
What should the tool do when it encounters a resource it cannot price (unsupported type, missing data, incomplete attributes)?

A) Skip it silently and only price what it can

B) Print it in a "not estimated" section with the reason, still exit 0

C) Option B, plus a `--strict` flag that exits non-zero if anything could not be priced

X) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question 8
What is the primary intended use context? (Drives CI ergonomics and exit-code behavior.)

A) Local developer CLI use, run ad hoc

B) Local use + CI pipeline use (stable JSON, meaningful exit codes, optional cost-threshold gate that fails the build)

C) Option B, plus a mode that posts/writes a cost summary comment for pull requests

X) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question 9
Cost period and currency for displayed figures?

A) Monthly cost (730 hours) in USD, plus hourly rate where relevant

B) Monthly + hourly, USD only, with a `--period` flag (hour/month) 

C) Configurable currency as well as period

X) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question: Security Extensions
Should security extension rules be enforced for this project?

A) Yes — enforce all SECURITY rules as blocking constraints (recommended for production-grade applications)

B) No — skip all SECURITY rules (suitable for PoCs, prototypes, and experimental projects)

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question: Resiliency Extensions
Should the resiliency baseline be applied to this project?

**What this extension is.** Enabling it applies a set of **directional, design-time best practices** for building resilient systems, derived from the **AWS Well-Architected Framework (Reliability Pillar)**. It steers requirements, design, and code toward fault tolerance, high availability, observability, and recoverability.

**What this extension is NOT.** It does **not** make your workload production-ready, nor certify any availability/RTO/RPO target. It is a starting point, not a substitute for a formal AWS Well-Architected Review.

A) Yes — apply the resiliency baseline as directional best practices and design-time guidance (recommended for business-critical workloads)

B) No — skip the resiliency baseline (suitable for PoCs, prototypes, and experimental projects where rapid iteration matters more than reliability)

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question: Property-Based Testing Extension
Should property-based testing (PBT) rules be enforced for this project?

A) Yes — enforce all PBT rules as blocking constraints (recommended for projects with business logic, data transformations, serialization, or stateful components)

B) Partial — enforce PBT rules only for pure functions and serialization round-trips (suitable for projects with limited algorithmic complexity)

C) No — skip all PBT rules (suitable for simple CRUD applications, UI-only projects, or thin integration layers with no significant business logic)

X) Other (please describe after [Answer]: tag below)

[Answer]:B
