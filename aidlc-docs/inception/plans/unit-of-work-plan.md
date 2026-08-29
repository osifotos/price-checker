# Unit of Work Plan — Price Checker

**Inputs**: approved `requirements.md`, `stories.md`, `execution-plan.md`, and the Application Design artifacts.
**Context**: greenfield, single deployable artifact (one static Go binary). "Units of work" here are **development units** for the per-unit CONSTRUCTION loop, not separate services. The Application Design already sketched six units (U1–U6) + the shared `pkg/schema`; these questions confirm or adjust that decomposition.

---

## Part A — Decomposition Questions

Fill in every `[Answer]:` tag. Use `X) Other` with a description if none fit.

### Question 1 — Overall decomposition model
The design proposes one binary composed of 6 development units built in dependency order (U1 plan-ingest → U2 pricing-core → U3 cost-engine → U4 output-renderers → U5 diff → U6 cli-app), plus `pkg/schema` written first.

A) Confirm: 6 units + a small preliminary `U0 schema` unit for `pkg/schema` (7 total)

B) Confirm the 6 units, fold `pkg/schema` into U1 (no separate schema unit)

C) Fewer, coarser units (e.g. merge into 3: ingest+pricing, engine+renderers+diff, cli)

D) More, finer units (e.g. split U2 pricing-core into "client+cache" and "catalog+pricers")

X) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2 — U4 renderers and U5 diff
U4 (table/JSON/HTML/github-comment renderers) and U5 (diff engine + diff renderers) both depend only on `pkg/schema` and are closely related.

A) Keep U4 and U5 separate (diff is its own unit with its own functional design)

B) Merge U4 + U5 into one "output & diff" unit

X) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 3 — Build sequencing
How should the per-unit CONSTRUCTION loop proceed?

A) Strictly sequential in dependency order; each unit fully designed + coded + unit-tested before the next starts (matches the AI-DLC per-unit loop default)

B) Sequential for U1→U2→U3, then U4 and U5 treated as a parallel pair, then U6

C) Sequential, but allow a thin "walking skeleton" of U6 (CLI wiring + `version`) early so end-to-end smoke tests exist sooner

X) Other (please describe after [Answer]: tag below)

[Answer]: C

### Question 4 — Per-unit "definition of done"
What must each unit satisfy before the loop moves on?

A) Compiles; exported interfaces implemented; example-based unit tests pass; `go vet` + `gofmt` clean

B) Option A plus the unit's PBT targets (where the Partial set applies) and recorded-fixture tests for any external-API mapping

C) Option B plus a short per-unit README/gauge of which stories it now satisfies

X) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 5 — AWS pricer coverage within U2
The v1 catalog has ~16 AWS resource groups. How should U2 sequence them?

A) Implement all ~16 pricers in U2 before U2 is "done"

B) U2 delivers the client + cache + catalog framework + a first slice of pricers (EC2, EBS, RDS, S3, Lambda, ALB/NLB, NAT Gateway); the remaining pricers (EKS, ElastiCache, DynamoDB, CloudWatch, EIP, snapshots, data transfer) are a tracked follow-on within U2 before Build & Test

C) Split pricers across two units by priority (core drivers vs the rest)

X) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6 — Repository / module setup
For the Go module and repo:

A) Single Go module at repo root, module path `github.com/<you>/price-checker` — tell me the path to use

B) Single module, placeholder module path `github.com/example/price-checker` for now, easy to rename later

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Part B — Execution Checklist (runs after plan approval)

- [x] B1. Load approved answers + Application Design artifacts
- [x] B2. Generate `aidlc-docs/inception/application-design/unit-of-work.md` — per unit: id, name, responsibility, packages, components owned, in-scope stories, definition of done, code-organization strategy (greenfield directory layout)
- [x] B3. Generate `aidlc-docs/inception/application-design/unit-of-work-dependency.md` — unit dependency matrix, build order, parallelization notes, integration points
- [x] B4. Generate `aidlc-docs/inception/application-design/unit-of-work-story-map.md` — every one of the 30 stories mapped to exactly one owning unit (+ supporting units), with a coverage check
- [x] B5. Validate: unit boundaries align with the package import rules in `component-dependency.md`; no story is unassigned; no unit has a cyclic dependency
- [x] B6. Content validation per `common/content-validation.md`
- [x] B7. Update `aidlc-state.md`; log approval prompt in `audit.md`; present completion message

---

## Mandatory Artifacts
- [x] `aidlc-docs/inception/application-design/unit-of-work.md`
- [x] `aidlc-docs/inception/application-design/unit-of-work-dependency.md`
- [x] `aidlc-docs/inception/application-design/unit-of-work-story-map.md`
