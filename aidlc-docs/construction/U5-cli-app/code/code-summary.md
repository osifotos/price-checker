# U5 `cli-app` — Code Summary

**Locations**: `internal/config/`, `internal/cli/`, `cmd/price-checker/`, `.github/workflows/`, `README.md`

> Go toolchain absent; `go mod tidy` + build/test run in Build and Test.

## Dependency added

`github.com/urfave/cli/v2` (Q2). TTY detection is dependency-free (`os.File.Stat` + `ModeCharDevice`), so `golang.org/x/term` was **not** added.

## Files

### `internal/config/`
| File | Contents |
|---|---|
| `config.go` | `Config` (20 settings), `Source`/`Provenance`, `Resolve(Inputs)` — layered **flag/env > config file > default** with per-field provenance; YAML config file (`price-checker.yml` or `--config`); unknown keys → warnings (not fatal); bad YAML → error |

### `internal/cli/`
| File | Contents |
|---|---|
| `run.go` | `Run(args, stdin, stdout, stderr) int` → `runWithDeps`; maps `cli.ExitCoder` to its code, other errors → exit 1; `noColorForced` (NO_COLOR env or non-TTY stdout) |
| `deps.go` | `Estimator` interface (`*estimator.Estimator` satisfies it), `Deps` test seam (`NewEstimator` factory, `Loader`/`Parser`), `newLogger` (slog → stderr, level from config) |
| `wiring.go` | `buildEstimator` — `awsauth.Load` → `awspricing.NewFromConfig` → `pricing.NewPriceListClient` → `pricing.NewCachingClient` → `estimator.New(aws.NewCatalog(), …)`; `renderOptions` |
| `app.go` | `env`, `buildApp` (urfave command tree: default/`breakdown`, `diff`, `report`, `usage generate`, `version`), `globalFlags` (21 flags, all with `PRICE_CHECKER_*` env vars), `resolveConfig`, custom help template with an **EXIT CODES** block |
| `commands.go` | the five actions; `writeRendered` (buffer → stdout or `--out` file, logs to stderr); `exitFor` (Outcome → `cli.Exit("", code)`); `estimateOrLoad` (prior-JSON vs estimate); debug `logProvenance` |
| `input.go` | `Classify` → `Stdin` / `TerraformDir` / `PlanFile` / `PriorJSON` (sniffs `"format_version"` vs `"schema_version"`) |
| `exit.go` | exit-code constants + `Evaluate(Outcome, Config)` — precedence `1 > 2 > 3` |
| `version.go` | `version`/`commit`/`date` vars (set via `-ldflags`), `versionString()` incl. `runtime.Version()` |

### `cmd/price-checker/main.go`
`func main() { os.Exit(cli.Run(os.Args, os.Stdin, os.Stdout, os.Stderr)) }`

### `.github/workflows/`
- `ci.yml` — gofmt, `go vet`, `go test`, golangci-lint, 5-target cross-compile (`CGO_ENABLED=0`)
- `release.yml` — on `v*` tag, build 5 static binaries with version stamp, attach to release

### `README.md`
Install, usage, all flags, **exit codes table**, usage-file schema + `usage generate`, JSON schema pointer, a **GitHub Actions cost-check recipe** (US-PR-COMMENT-3, `later`), minimum IAM policy, v1 scope.

## Tests

| File | Covers |
|---|---|
| `config/config_test.go` | precedence (flag > file > default) + provenance; no file → defaults; unknown key → warning; bad YAML → error |
| `cli/exit_test.go` | full `Evaluate` matrix incl. precedence `1 > 2 > 3` |
| `cli/input_test.go` | classify stdin/dir/plan-file/prior-JSON/other; missing file → error |
| `cli/run_test.go` | `version` (exit 0, no AWS); `breakdown` table on stdout; `--format json` valid JSON on stdout; `--strict` → exit 2; `--threshold-monthly` → exit 3 **with full output printed first**; unknown `--format` → exit 1 + stderr message; estimator error → exit 1; `--help` lists every flag + exit-code block |

## Stories advanced

- **US-CI-GATE-1** (stable exit codes 0/1/2/3, documented), **US-CI-GATE-2** (thresholds + `--strict`), **US-CONFIG-INSTALL-1** (config file + precedence + provenance), **US-CONFIG-INSTALL-2** (self-documenting CLI, stdout/stderr split, `version`), **US-CONFIG-INSTALL-3** (single static binary, 5-target CI, offline `version`), **US-PR-COMMENT-3** (`later` — recipe in README).

## Notes

- Global flags are duplicated on leaf commands (the standard urfave/cli v2 pattern) so `price-checker breakdown --path …` works as well as `price-checker --path …`.
- `HideVersion: true` frees `-v` for `--show-components`; `price-checker version` is the version command.
- `report` = `breakdown`/`diff` with `--format html` and a default `--out` file.
