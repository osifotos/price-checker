# Requirements Clarification Questions — Price Checker

Thanks for the answers. I found two points that need clarification before I can write the requirements document.

---

## Ambiguity 1: "All resources" scope (Q3)

Your Q3 answer was "All resources depending on what it scanned from the project folder", but the pricing source (AWS Price List API) and the effort of modelling each resource type mean a v1 realistically ships with a **defined catalog** of AWS resource types that get real cost estimates; anything outside the catalog would appear in the "not estimated" section (Q7).

Two things are actually being asked here, so please answer both sub-questions.

### Clarification Question 1a
How should the pricing catalog be scoped for v1?

A) Ship a fixed catalog of the common AWS cost drivers now (EC2, EBS, EBS snapshots, EIP, RDS instances + storage, S3 storage/requests, Lambda, ALB/NLB/ELB, NAT Gateway, EKS, ElastiCache, DynamoDB, CloudWatch, data transfer), and grow it later. Everything else -> "not estimated".

B) Start with a smaller catalog (EC2, EBS, RDS, S3, Lambda, ALB/NLB, NAT Gateway) and grow it later. Everything else -> "not estimated".

C) Architect a generic, data-driven pricing-rule engine so new resource types can be added via config/rule files without code changes, and seed it with the Option A catalog.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

### Clarification Question 1b
The tool prices resources found in the Terraform plan. Should it also flag every resource in the plan that it did NOT price, so you can see coverage gaps?

A) Yes — always list unpriced in-plan resources in the "not estimated" section with a reason

B) Yes, but only when a `--show-skipped` flag is passed

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Ambiguity 2: Configurable currency (Q9)

The AWS Price List API returns prices in **USD only**. "Configurable currency" therefore requires converting USD to the target currency using an exchange rate the tool has to get from somewhere.

### Clarification Question 2
How should non-USD currency be handled in v1?

A) Drop configurable currency for v1 — USD only, with a configurable `--period` (hour/month). Revisit currency later.

B) Keep configurable currency, using a user-supplied fixed rate (e.g. `--fx-rate EUR=0.92`); no network FX lookup

C) Keep configurable currency with an automatic FX rate fetched from a public exchange-rate API, `--fx-rate` overrides

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Note (no answer needed): v1 scope size

Your selections include an HTML report (Q2:D) and a PR-comment mode (Q8:C) on top of the core estimator, diff view, JSON output, usage-file support, and on-disk caching. That is a sizeable v1. I will plan it as multiple units of work and we can sequence/defer pieces during Workflow Planning — flagging now so the size is visible.
