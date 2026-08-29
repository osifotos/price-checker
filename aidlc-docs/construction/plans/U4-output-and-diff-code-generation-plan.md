# U4 `output-and-diff` — Code Generation Plan (consolidated cadence)

Code: `internal/render/`, `internal/render/html/`, `internal/diff/`. No new deps.
Stories: US-LOCAL-ESTIMATE-3/4, US-MACHINE-OUTPUT-1/2, US-REPORT-1, US-DIFF-1/2, US-PR-COMMENT-1/2.

| # | Step | Status |
|---|---|---|
| 1 | `renderer.go` (Options, interfaces, RendererFor) | [x] |
| 2 | `viewmodel.go` (money formatters, palette, htmlVM) | [x] |
| 3 | `table.go` (breakdown + diff) | [x] |
| 4 | `json.go` (breakdown + diff) | [x] |
| 5 | `gh.go` (breakdown + diff Markdown) | [x] |
| 6 | `html.go` + `html/report.{html.tmpl,css,js}` (embedded, self-contained) | [x] |
| 7 | `internal/diff/differ.go` (pure Diff) | [x] |
| 8 | Tests: render (all formats, schema-validate JSON, HTML self-contained, gh deterministic), diff (classification + PBT-03), import-lint x2 | [x] |
| 9 | `code-summary.md` | [x] |

Build/lint/test deferred to Build and Test.
