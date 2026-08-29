# Application Design Plan — Price Checker

**Role**: Software architect / designer
**Inputs**: `requirements.md`, `stories.md`, `personas.md`, `execution-plan.md` (all approved)
**Scope**: High-level components, interfaces, service orchestration, dependencies. Detailed business logic is deferred to per-unit Functional Design.

---

## Part A — Design Questions

Fill in every `[Answer]:` tag. Use `X) Other` with a description if none fit.

### Question 1 — Go project layout
Which source layout should the codebase use?

A) Standard layout: `cmd/price-checker/` (main), `internal/` for all packages (plan, pricing, engine, render, diff, cli, config), `testdata/` for fixtures

B) Flat `pkg/` layout with exported packages (allows external reuse as a library)

C) Hybrid: `cmd/` + `internal/` for app code, plus a small exported `pkg/schema` for the public JSON/usage types only

X) Other (please describe after [Answer]: tag below)

[Answer]: C

### Question 2 — CLI framework
Which CLI library?

A) `spf13/cobra` — subcommands, help generation, flag binding; the de-facto standard, pairs with `spf13/viper`

B) `urfave/cli/v2` — lighter, simpler API

C) Standard library `flag` only — zero dependencies, more manual work for subcommands

X) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 3 — Configuration precedence implementation
FR-12.3 requires flag > env > config file > default. How to implement it?

A) `spf13/viper` (bound to cobra flags) — handles env, config file, defaults automatically

B) Hand-rolled resolver: a `Config` struct, explicit layering, ~one file — no extra dependency, fully explicit precedence and "where did this value come from" reporting

C) `koanf` — modular config loader, lighter than viper

X) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 4 — Decimal money type
NFR-2.1 / FR-6.4 require decimal arithmetic (no float accumulation). Which type?

A) `shopspring/decimal` — widely used, arbitrary precision, simple API

B) `cockroachdb/apd` — IEEE-754 decimal, explicit context/rounding, higher performance

C) Standard library `math/big.Rat` — exact rationals, no dependency, awkward formatting

X) Other (please describe after [Answer]: tag below)

[Answer]: C

### Question 5 — Pricer plugin interface & registration
NFR-5.1 requires adding a resource type by implementing one interface and registering it. Preferred registration mechanism?

A) Central registry map populated by `init()` in each pricer file (`register("aws_instance", &instancePricer{})`); the catalog is "whatever registered itself"

B) Explicit slice/list built in one `catalog.go` constructor (`NewCatalog()` returns all pricers) — no `init()` magic, deterministic, easy to test a subset

C) Registry keyed by a match function (not just resource type string) so one pricer can claim several related types or match on attributes

X) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 6 — Pricing layers
How should the AWS pricing side be layered?

A) Two layers: (1) `PriceListClient` — thin wrapper over AWS SDK `GetProducts`, returns raw price dimensions for a service+filters query; (2) `Pricer` per resource type — builds queries, picks dimensions, emits cost components. Cache is a decorator around layer (1).

B) One layer: each `Pricer` talks to the AWS SDK directly and does its own caching

C) Three layers: add a `PriceCatalog` in the middle that pre-resolves and holds all prices a run needs, so `Pricer`s are pure functions over an in-memory price set

X) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 7 — Output renderer abstraction
How should the four output formats be structured?

A) One `Renderer` interface (`Render(w io.Writer, result *Breakdown) error`) with four implementations (table, json, html, github-comment); diff rendering is a second interface `DiffRenderer`

B) A single renderer that switches on format internally

C) `Renderer` interface plus a shared "view model" struct that pre-formats numbers/labels once, so each renderer is presentation-only

X) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8 — Top-level orchestration (service layer)
How should the end-to-end flow be organised?

A) Two service types: `Estimator` (parse plan → price resources → aggregate → produce `Breakdown`) and `Differ` (two `Breakdown`s → `DiffResult`); the CLI layer wires inputs/outputs and applies exit-code/threshold policy

B) One `App` service with methods `Breakdown()`, `Diff()`, `GenerateUsage()`; CLI is a thin arg parser

C) Pipeline of stages (`[]Stage`) the CLI composes per subcommand

X) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 9 — HTML report templating
FR-10 needs a self-contained HTML report (inline CSS/JS, no external requests).

A) Go `html/template` with the template + CSS embedded via `//go:embed`; no JS, or tiny inline vanilla JS for collapse/expand

B) Build the HTML with a small builder in code (no template files)

C) `html/template` + a charting/JS library inlined from `//go:embed` for visual breakdowns

X) Other (please describe after [Answer]: tag below)

[Answer]: C

### Question 10 — Public JSON schema definition
The `--format json` output is a consumer contract (FR-8).

A) Go structs with `json` tags as the source of truth; a `schema_version` constant; a golden-file test pins the shape; document it in README

B) Author a JSON Schema file, `//go:embed` it, validate output against it in tests, publish it in the repo

C) Both — structs are canonical, plus a generated JSON Schema committed for consumers

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Part B — Execution Checklist (runs after plan approval)

- [x] B1. Load approved requirements, stories, execution plan, and these answers
- [x] B2. `components.md` — each component: name, purpose, responsibilities, public interface (Go interface sketch), which unit (U1–U6) owns it
- [x] B3. `component-methods.md` — method signatures + input/output types + one-line purpose per component (no business rules yet)
- [x] B4. `services.md` — `Estimator` / `Differ`: responsibilities, orchestration sequence, error/not-estimated aggregation, policy boundary with the CLI
- [x] B5. `component-dependency.md` — dependency matrix, communication patterns, data-flow diagram (mermaid + text alternative), build order
- [x] B6. `application-design.md` — consolidation of the above with the key design decisions (Q1–Q10) and their rationale
- [x] B7. Validate completeness: every FR/NFR and every unit has an owning component; every story's behaviour has a home
- [x] B8. Content validation per `common/content-validation.md` (mermaid syntax checked; ASCII diagrams use plain `+ - | < > v` only)
- [x] B9. Update `aidlc-state.md`; log approval prompt in `audit.md`; present completion message

---

## Mandatory Artifacts
- [x] `aidlc-docs/inception/application-design/components.md`
- [x] `aidlc-docs/inception/application-design/component-methods.md`
- [x] `aidlc-docs/inception/application-design/services.md`
- [x] `aidlc-docs/inception/application-design/component-dependency.md`
- [x] `aidlc-docs/inception/application-design/application-design.md`
