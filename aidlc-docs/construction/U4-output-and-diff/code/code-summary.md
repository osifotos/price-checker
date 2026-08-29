# U4 `output-and-diff` — Code Summary

**Locations**: `internal/render/`, `internal/render/html/`, `internal/diff/`
**Imports**: `pkg/schema` + stdlib **only** (lint-enforced in both packages). No new dependencies.

> Go toolchain absent; build/lint/test run in Build and Test.

## Files

### `internal/render/`
| File | Contents |
|---|---|
| `renderer.go` | `Options`, `Renderer`, `DiffRenderer`, `RendererFor(format)` → `(Renderer, DiffRenderer, error)` for `table`/`json`/`html`/`github-comment` (unknown → error) |
| `viewmodel.go` | money formatters `money2` (`$1,234.56`), `rate4` (`$0.0416`), `signed2` (`+$12.34`), thousands `group`; ANSI `palette` (empty when `Color=false`); `htmlVM` + `breakdownVM`/`diffVM` builders |
| `table.go` | `tableRenderer` — `text/tabwriter` columns, `PROJECT TOTAL`, coverage line, `NOT ESTIMATED (n)` section; HOURLY column shown for `--period hour` or `-v`; ASCII-only component prefix `  - `; diff variant with `+/-` deltas |
| `json.go` | `jsonRenderer` — `json.Encoder` indent 2, `SetEscapeHTML(false)`; exact `pkg/schema` marshalling for breakdown and diff |
| `gh.go` | `ghRenderer` — Markdown: headline total / delta, compact table, `> :warning:` not-estimated note, `<details>` full breakdown, `<sub>` version/timestamp; deterministic |
| `html.go` | `htmlRenderer` — `html/template` + `//go:embed` of `.tmpl`/`.css`/`.js`; assets injected as `template.CSS` / `template.JS`; contextual auto-escaping for resource names |
| `html/report.html.tmpl`, `report.css`, `report.js` | self-contained report: header + totals + metadata, by-module and by-resource tables, expandable component rows (4-line vanilla JS), not-estimated table, diff branch. **Zero external URLs.** |

### `internal/diff/`
| File | Contents |
|---|---|
| `differ.go` | `Differ.Diff(base, proposed)` — pure function; index by address, classify added/removed/changed/unchanged, carry `NotEstimated` (no fabricated delta), `big.Rat` totals, `Canonicalize` (order by |delta| desc) |

## Tests

| File | Covers |
|---|---|
| `render/render_test.go` | `RendererFor` all formats + unknown; table (total line, not-estimated section, **no ANSI when `Color=false`**, component indent); JSON **validates against `schema.BreakdownSchemaJSON`** + faithful round trip + `SetEscapeHTML(false)`; HTML **self-contained** (no `http://`/`https://`/`src="//"`) + hostile resource name escaped; github-comment deterministic + headline + `<details>` + not-estimated note; all four diff renderers non-empty |
| `render/imports_test.go` | renderers import only stdlib + `pkg/schema` |
| `diff/differ_test.go` | classification matrix; total delta == Σ estimated deltas; canonical order (largest \|delta\| first); NotEstimated carried with `delta = 0`; **PBT-03** (`rapid`) Σ estimated deltas == total delta |
| `diff/imports_test.go` | differ imports only stdlib + `pkg/schema` |

## Stories advanced

- **US-LOCAL-ESTIMATE-3/4** (table + components + period), **US-MACHINE-OUTPUT-1** (JSON = schema, schema-validated), **US-MACHINE-OUTPUT-2** (renderers take `io.Writer` — `--out` is U5), **US-REPORT-1** (self-contained HTML), **US-DIFF-1/2** (Differ + all four diff formats), **US-PR-COMMENT-1/2** (Markdown headline delta + sorted-by-|delta| table + collapsible detail).

## Notes

- `RendererFor` + `Options` (period, `-v`, `NO_COLOR`/TTY) are wired to flags in U5; diff input resolution (plan vs prior-JSON vs `--from-plan`) is U5 + Estimator.
- The HTML report deliberately ships one tiny embedded JS file; no charting library was added (kept the binary lean, per the design note).
