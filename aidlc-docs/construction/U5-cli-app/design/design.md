# U5 `cli-app` — Consolidated Design

**Unit**: U5 · **Packages**: `internal/config`, `internal/cli`, `cmd/price-checker`, `.github/workflows`
**Depends on**: U0–U4 (all)
**Owns**: US-CI-GATE-1/2, US-CONFIG-INSTALL-1/2/3, US-PR-COMMENT-3 (docs, `later`)

---

## 1. Functional Design

### 1.1 `internal/config` (Q3=B: hand-rolled resolver)

```go
type Config struct {
    Path            string
    CompareTo       string
    FromPlan        bool
    Format          string        // table|json|html|github-comment
    Out             string
    Period          string        // month|hour
    UsageFile       string
    AWSRegion       string
    Profile         string
    CacheDir        string
    CacheTTL        time.Duration
    NoCache         bool
    RefreshCache    bool
    Strict          bool
    ThresholdMonthly     float64   // 0 = disabled
    ThresholdDiffMonthly float64
    ShowComponents  bool
    Concurrency     int
    LogLevel        string        // error|warn|info|debug
    NoColor         bool
}

type Source string // "flag", "env", "file", "default"
type Provenance map[string]Source

func Resolve(in Inputs) (Config, Provenance, error)
```

`Inputs` carries, per setting: the flag/env value and whether it was set (`map[string]Raw`), plus the resolved config-file path. Precedence per key: **flag/env (set) > file > built-in default**. urfave/cli merges flag+env, so "set" means `ctx.IsSet(name)`.

- Config file: `price-checker.yml` in the working dir, or `--config <path>`. YAML with snake_case keys mirroring `Config`. Unknown keys → warning (returned), not error.
- `Provenance` records where each field's final value came from; `--log-level debug` prints the resolved config + provenance.
- `Format` default `table`; `Period` default `month`; `CacheTTL` default `168h`; `Concurrency` default `8`; `LogLevel` default `warn`.
- `NoColor` also true when the `NO_COLOR` env var is present or stdout is not a TTY (decided in `cli`, passed in).

### 1.2 `internal/cli`

**InputResolver**

```go
type InputKind int // PlanFile, TerraformDir, PriorJSON, Stdin
func Classify(value string) (InputKind, error)
```
- `""` / `"-"` → `Stdin`
- directory → `TerraformDir`
- file: sniff first non-space byte + look for `"format_version"` → `PlanFile`; else if it parses as a price-checker `schema.Breakdown` (`"schema_version"` present) → `PriorJSON`; else `PlanFile` (let the parser produce the error).

**App** (`urfave/cli/v2`)

| Command | Action |
|---|---|
| *(default)* / `breakdown` | Estimator.Estimate → Renderer.Render → threshold/strict → exit |
| `diff` | resolve `--path` (+`--compare-to` or `--from-plan`) → Differ → DiffRenderer.RenderDiff → threshold → exit |
| `report` | forces `--format html`, defaults `--out` to `price-checker-report.html` |
| `usage generate` | load+parse plan (no pricing) → `usage.GenerateSkeleton` → out |
| `version` | print `versionString()`; works offline; exit 0 |

Global flags (all with `PRICE_CHECKER_*` env vars): `--path/-p`, `--compare-to`, `--from-plan`, `--format/-f`, `--out/-o`, `--period`, `--usage-file`, `--aws-region`, `--profile`, `--config`, `--cache-dir`, `--cache-ttl`, `--no-cache`, `--refresh-cache`, `--strict`, `--threshold-monthly`, `--threshold-diff-monthly`, `--show-components/-v`, `--concurrency`, `--log-level`, `--no-color`.

**Wiring** (`buildEstimator`):
1. `cfg := awsauth.Load(ctx, cfg.Profile, cfg.AWSRegion)`
2. `raw := pricing.NewPriceListClient(awspricing.NewFromConfig(sdkCfg), pricing.ClientConfig{Concurrency: cfg.Concurrency, Logger: log})`
3. `cached := pricing.NewCachingClient(raw, pricing.CacheOptions{Dir: cfg.CacheDir, TTL: cfg.CacheTTL, NoCache: cfg.NoCache, Refresh: cfg.RefreshCache})`
4. `est := estimator.New(estimator.Deps{Loader: plan.NewLoader(), Parser: plan.NewParser(), Catalog: aws.NewCatalog(), Querier: cached, Region: pricing.RegionResolver{}, Agg: engine.Aggregator{}, Log: log, ToolVersion: version, Now: time.Now})`

**ExitPolicy**

```go
type Outcome struct {
    Err               error
    NotEstimatedCount int      // components + resources
    MonthlyTotal      float64
    MonthlyDelta      *float64  // nil for non-diff
}
func Evaluate(o Outcome, cfg config.Config) int
```
- `1` if `o.Err != nil` (runtime error)
- else `2` if `cfg.Strict && o.NotEstimatedCount > 0`
- else `3` if `cfg.ThresholdMonthly > 0 && o.MonthlyTotal > cfg.ThresholdMonthly`, or `cfg.ThresholdDiffMonthly > 0 && o.MonthlyDelta != nil && *o.MonthlyDelta > cfg.ThresholdDiffMonthly`
- else `0`
- Precedence `1 > 2 > 3`. Full output is always written **before** returning a non-zero code for 2/3.

**Logging** — `slog` text handler on **stderr**; level from `cfg.LogLevel`. stdout carries only rendered output (or the `--out` file).

**version** — `var version, commit, date = "dev", "none", "unknown"` set via `-ldflags`; `versionString()` includes `runtime.Version()`.

### 1.3 `cmd/price-checker/main.go`

```go
func main() { os.Exit(cli.Run(os.Args, os.Stdin, os.Stdout, os.Stderr)) }
```
`cli.Run` builds the app, runs it, maps any `cli.ExitCoder` to its code, prints other errors to stderr and returns 1.

### 1.4 `.github/workflows`

- `ci.yml`: `go vet`, `gofmt -l`, `go test ./...` (with `-count=1`), `golangci-lint`, cross-compile matrix (5 targets).
- `release.yml`: on tag, build the 5 static binaries (`CGO_ENABLED=0`, `-ldflags` version stamp), attach to the GitHub release.
- README section (US-PR-COMMENT-3, `later`): a documented GitHub Actions job that assumes a role, generates base+PR plans, runs `price-checker diff --format github-comment`, posts/updates the comment, and gates on exit 3.

---

## 2. NFR Requirements

| NFR | Obligation |
|---|---|
| NFR-4.1 | missing-credential error from the first Price List call → wrapped by U2 with the actionable message; CLI surfaces it as exit 1 |
| NFR-4.2 | `--help` lists every flag; exit codes + usage-file schema documented in README |
| NFR-4.3 | table output honours `NO_COLOR` and non-TTY |
| NFR-6.1/6.2 | `CGO_ENABLED=0`; 5-target build in CI; only optional runtime dep is `terraform` (dir mode) |
| NFR-7.1 | `--log-level debug` logs each Price List query + cache hit/miss (emitted by U2 via the injected logger) + the resolved config/provenance |
| FR-11.1 | exit codes `0/1/2/3`, stable, documented |

### 2.1 Dependencies

`github.com/urfave/cli/v2` (new). `golang.org/x/term` for TTY detection (small; or use a build-tag-free `os.Stdout.Stat()` check — **chosen: `term.IsTerminal(int(os.Stdout.Fd()))`** via `golang.org/x/term`). `gopkg.in/yaml.v3` (already present) for the config file.

## 3. NFR Design

- **`cli.Run` returns an int** — the single place process exit is decided; `main` just calls `os.Exit`.
- **Services get `io.Writer`s** — nothing in `cli` writes to `os.Stdout` directly except the final copy of the render buffer (so `--out` is a one-line swap).
- **Estimator built lazily** — `version` and `usage generate` never construct the AWS client, so they work offline.
- **TTY / NO_COLOR** resolved once in `cli.Run` and passed down as `cfg.NoColor`.
- **Threshold comparison** uses the float totals from the rendered `schema.Breakdown`/`DiffResult` (`strconv.ParseFloat` of the decimal strings) — acceptable for a gate check; the displayed numbers stay decimal.

### 3.1 File plan

```
internal/config/config.go        Config, Source, Provenance, Resolve, file load
internal/cli/run.go              Run(args, stdin, stdout, stderr) int
internal/cli/app.go              buildApp (urfave command tree + flags)
internal/cli/commands.go         breakdown / diff / report / usage / version actions
internal/cli/input.go            InputResolver.Classify
internal/cli/wiring.go           buildEstimator, buildRenderer
internal/cli/exit.go             Outcome, Evaluate
internal/cli/version.go          version vars + versionString
cmd/price-checker/main.go
.github/workflows/ci.yml, release.yml
*_test.go
```

### 3.2 Test targets (DoD)

- `config`: precedence (flag > env > file > default) with provenance assertions; unknown file key → warning; missing file → defaults; bad YAML → error.
- `cli/input`: classify stdin / dir / plan file / prior-JSON file / non-JSON file.
- `cli/exit`: full matrix — err→1; strict+notEstimated→2; over threshold→3; diff threshold→3; precedence 1>2>3; nothing→0.
- `cli/run`: end-to-end with a fake estimator/renderer injected via a test seam:
  - `version` → exit 0, prints version, no AWS
  - `breakdown` with a plan-JSON fixture on stdin + stub querier → table on stdout, exit 0
  - `--strict` with a not-estimated resource → exit 2
  - `--threshold-monthly` exceeded → exit 3, full output still printed
  - `--format json` → valid JSON on stdout, logs on stderr
  - unknown `--format` → exit 1
- `cli`: `--help` contains every flag name; exit codes documented string present.
- import: `cmd/price-checker/main.go` is ≤ a few lines.

---

## 4. Post-U5

- **Build and Test** stage: `go mod tidy`, run everything, the 5-target build, wire CI.
- README: flags, exit codes, usage-file schema, JSON schema pointer, GitHub Actions recipe.
