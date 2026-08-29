# U4 `output-and-diff` — Consolidated Design

**Unit**: U4 · **Packages**: `internal/render`, `internal/render/html`, `internal/diff`
**Depends on**: U0 `pkg/schema` **only** (NFR-5.3) · SDK-free, plan-free
**Owns**: US-LOCAL-ESTIMATE-3/4, US-MACHINE-OUTPUT-1/2, US-REPORT-1, US-DIFF-1/2, US-PR-COMMENT-1/2

---

## 1. Functional Design

### 1.1 Interfaces (`internal/render`)

```go
type Options struct {
    Period         schema.Period // month | hour
    ShowComponents bool          // -v: indent per-resource components
    Color          bool          // false when NO_COLOR / non-TTY
}
type Renderer interface {
    Render(w io.Writer, b *schema.Breakdown, opts Options) error
}
type DiffRenderer interface {
    RenderDiff(w io.Writer, d *schema.DiffResult, opts Options) error
}

// RendererFor resolves --format to the pair (some formats implement both).
func RendererFor(format string) (Renderer, DiffRenderer, error)
//   "table"          -> tableRenderer{}
//   "json"           -> jsonRenderer{}
//   "html"           -> htmlRenderer{}
//   "github-comment" -> ghRenderer{}
```

- The producer (U3 Estimator / U5) calls `Canonicalize()` before handing a doc to a renderer; renderers do **not** re-sort.
- Renderers write to the given `io.Writer` and return any write/encode error. They never call `os.Exit`, read flags, or touch the clock.

### 1.2 `tableRenderer` (US-LOCAL-ESTIMATE-3/4, FR-7)

Breakdown:
```
 NAME                              MONTHLY      HOURLY
 aws_instance.web                  $30.37       $0.0416
   ├─ Instance usage (t3.medium)   $30.37       $0.0416     (only with ShowComponents)
 module.net.aws_nat_gateway.this   $32.85       $0.0450
 ─────────────────────────────────────────────────────
 PROJECT TOTAL                     $63.22       $0.0866

 NOT ESTIMATED (2)
 aws_s3_bucket.assets  Storage  NO_USAGE_DATA  no usage value was supplied for this component
 aws_kms_key.main               UNSUPPORTED_TYPE  resource type is not in the pricing catalog
```
- Money formatted to 2 dp with `$` for monthly; 4 dp for hourly (rates are small).
- `Color` true → totals bold, not-estimated section header yellow (ANSI); false → plain.
- Fits 100 cols; long addresses are not truncated (wrap by terminal).
- The HOURLY column is shown only when `Period == hour` **or** `ShowComponents`; MONTHLY always.
- ASCII connectors only (`├─`, `└─` are box-drawing → use `- ` prefix to stay ASCII-safe per repo ASCII rules): components use `  - ` prefix.

Diff:
```
 NAME                    PRIOR     NEW       CHANGE
 aws_nat_gateway.this    $0.00     $32.85    +$32.85   (added)
 aws_instance.web        $30.66    $61.32    +$30.66
 aws_eip.old             $3.65     $0.00     -$3.65    (removed)
 ──────────────────────────────────────────────────
 TOTAL CHANGE                                +$59.87 / month
```

### 1.3 `jsonRenderer` (US-MACHINE-OUTPUT-1/2, FR-8)

- `Render`: `enc := json.NewEncoder(w); enc.SetIndent("", "  "); enc.SetEscapeHTML(false); enc.Encode(b)`.
- `RenderDiff`: same for `*schema.DiffResult`.
- Output is exactly the `pkg/schema` marshalling (deterministic; validated against the embedded JSON Schema in U0's tests and re-checked here with a golden test).
- `Period` / `ShowComponents` / `Color` ignored (JSON always carries everything).

### 1.4 `htmlRenderer` (US-REPORT-1, FR-10)

- `html/template` parsed from `//go:embed`ed `internal/render/html/report.html.tmpl`; CSS embedded from `report.css`; a ~small inline JS snippet (embedded `report.js`, < 4 KB) only for collapse/expand of the component rows.
- **Zero external requests**: no `<link href="http…">`, no `<script src="http…">`, no web-font URLs, no external images.
- Sections: header (totals + metadata), per-module table, per-resource rows with expandable components, "not estimated" table, and — when `RenderDiff` — the change table.
- `template.HTMLEscapeString` via `html/template`'s contextual auto-escaping protects against a hostile resource name.
- One template with a `{{if .IsDiff}}` branch; a `viewModel` struct pre-formats money strings so the template has no logic.

### 1.5 `ghRenderer` (US-PR-COMMENT-1/2, FR-11.3)

Breakdown:
```markdown
### 💰 price-checker — estimated monthly cost: **$63.22**

| Resource | Monthly |
|---|--:|
| `aws_instance.web` | $30.37 |
| `module.net.aws_nat_gateway.this` | $32.85 |

> ⚠️ 2 resources not estimated

<details><summary>Full breakdown</summary>

… table with components …

</details>

<sub>price-checker v0.0.0 · 2026-08-29T00:00:00Z</sub>
```

Diff — headline is the delta:
```markdown
### 💰 price-checker — estimated monthly cost change: **+$59.87** ($34.31 → $94.17)

| Resource | Prior | New | Change |
|---|--:|--:|--:|
| `aws_nat_gateway.this` | $0.00 | $32.85 | **+$32.85** |
…
```
- Deterministic (schema is canonicalized) → a bot can update-in-place.
- Changed resources sorted by |delta| desc (already done by `DiffResult.Canonicalize`).

### 1.6 `internal/diff` — Differ (US-DIFF-1/2, FR-9)

```go
type Differ struct{}
func (Differ) Diff(base, proposed *schema.Breakdown) *schema.DiffResult
```

Algorithm:
1. Index `base.Resources` and `proposed.Resources` by `Address`.
2. For each address in the union (sorted):
   - proposed only → `ResourceDelta{kind: added, prior: 0, new: cost}`
   - base only → `ResourceDelta{kind: removed, prior: cost, new: 0}`
   - both, `MonthlyCost` differs → `changed`
   - both, equal → `unchanged`
   - if either side's resource has **zero components** (i.e. was not estimated), set `NotEstimated: true` and do not compute a delta (`delta_monthly = "0"`).
3. `total_delta_monthly` = Σ over deltas of (`new` − `prior`) for **estimated** rows; `total_prior/new_monthly` from the two breakdowns' totals.
4. `Summary` = a merge (resources_total = union size; estimated = rows not flagged NotEstimated).
5. `Metadata` = `proposed.Metadata` with `GeneratedAt` kept.
6. `d.Canonicalize()` (orders by |delta| desc).

- `Diff` is a pure function; `big.Rat` for all arithmetic.
- A resource unchanged and not-estimated on both sides → `unchanged`, `NotEstimated: true`.

---

## 2. NFR Requirements

| NFR | Obligation |
|---|---|
| NFR-2.1 | diff arithmetic in `big.Rat` |
| NFR-2.3 (PBT) | PBT-02: JSON round trip of a rendered breakdown/diff equals the input; PBT-03: diff invariant — Σ deltas == total_new − total_prior (estimated rows), delta signs match |
| NFR-3.3 | renderers emit only cost-relevant fields already in `schema`; HTML/gh output contain no attribute values |
| NFR-5.3 | packages import `pkg/schema` + stdlib only (import-lint) |
| NFR-6 | `html/template`, `encoding/json`, `//go:embed` — all stdlib |
| FR-10.1 | HTML asset-inlining test: rendered output contains no `http://`/`https://` and no `src=`/`href=` to an external scheme |

### 2.1 Dependencies

**None.** stdlib only.

---

## 3. NFR Design

- **`viewModel`** builders (`internal/render/viewmodel.go`) turn a `*schema.Breakdown`/`*schema.DiffResult` + `Options` into a presentation struct (formatted money, chosen columns). Table, HTML, and gh renderers consume the view model; JSON does not. This keeps money formatting in exactly one place per unit and the HTML template logic-free.
- **Money formatting**: `money2(s)` → `$1,234.56`; `rate4(s)` → `$0.0416`; `signed2(s)` → `+$12.34` / `-$12.34` / `$0.00`. All parse the decimal string with `schema.StringToRat` and never touch float.
- **Color**: a `palette` with `bold`/`dim`/`warn`/`reset`; when `Color==false` every code is `""`.
- **HTML**: `template.Must(template.New("report").Parse(embedded))`; executed once per call; output buffered then written.
- **Determinism**: renderers iterate the already-canonicalized slices in order; maps are never ranged for output.

### 3.1 File plan

```
internal/render/doc.go
internal/render/renderer.go     Options, Renderer, DiffRenderer, RendererFor
internal/render/viewmodel.go    viewModel builders + money formatters + palette
internal/render/table.go        tableRenderer
internal/render/json.go         jsonRenderer
internal/render/gh.go           ghRenderer
internal/render/html.go         htmlRenderer (parses embedded assets)
internal/render/html/report.html.tmpl
internal/render/html/report.css
internal/render/html/report.js
internal/diff/differ.go         Differ.Diff
*_test.go + testdata/golden/*
```

### 3.2 Test targets (DoD)

- `table_test.go`: golden files for breakdown (with/without components, month/hour) and diff; `Color=false` has no ESC bytes; totals correct.
- `json_test.go`: output validates against `schema.BreakdownSchemaJSON` / `DiffSchemaJSON`; round-trips; `SetEscapeHTML(false)` (no `<`).
- `gh_test.go`: headline shows total / delta; `<details>` present; not-estimated note when count > 0; deterministic across 2 runs.
- `html_test.go`: renders without error for breakdown + diff; **contains no `http://` / `https://`**; no `src="//"`; component rows present; hostile resource name is escaped.
- `differ_test.go`: added / removed / changed / unchanged classification; not-estimated carried; total delta == new − prior; single-plan-derived via two breakdowns; **PBT-03** (`rapid`) — Σ estimated deltas == total delta.
- `render_test.go`: `RendererFor` returns the right pair for each format; unknown format → error.
- `imports_test.go`: `internal/render*` and `internal/diff` import only `pkg/schema` + stdlib.

---

## 4. Open items → U5

- `RendererFor` + `Options` are wired to `--format`, `--period`, `-v/--show-components`, `NO_COLOR`/TTY detection in U5.
- `--out` file writing is U5 (renderers just take an `io.Writer`).
- Diff input resolution (plan vs prior-JSON vs `--from-plan`) is U5's `InputResolver` + Estimator.
