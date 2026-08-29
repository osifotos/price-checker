# U5 `cli-app` — Code Generation Plan (consolidated cadence)

Code: `internal/config/`, `internal/cli/`, `cmd/price-checker/`, `.github/workflows/`, `README.md`.
Dep added: `github.com/urfave/cli/v2`.
Stories: US-CI-GATE-1/2, US-CONFIG-INSTALL-1/2/3, US-PR-COMMENT-3 (later, docs).

| # | Step | Status |
|---|---|---|
| 1 | `internal/config/config.go` (Config, Resolve, provenance, file layer) | [x] |
| 2 | `internal/cli/version.go`, `exit.go`, `input.go` | [x] |
| 3 | `internal/cli/deps.go` (Estimator iface, Deps seam, logger) | [x] |
| 4 | `internal/cli/wiring.go` (buildEstimator, renderOptions) | [x] |
| 5 | `internal/cli/app.go` (urfave command tree, flags, help) | [x] |
| 6 | `internal/cli/commands.go` (5 actions, writeRendered, exitFor) | [x] |
| 7 | `internal/cli/run.go` (Run, runWithDeps, exit mapping) | [x] |
| 8 | `cmd/price-checker/main.go` | [x] |
| 9 | `.github/workflows/ci.yml`, `release.yml`; `README.md` | [x] |
| 10 | Tests: config precedence, exit matrix, input classify, run (version/table/json/strict/threshold/unknown-format/error/help) | [x] |
| 11 | `code-summary.md` | [x] |

Build/lint/test + `go mod tidy` deferred to Build and Test.
