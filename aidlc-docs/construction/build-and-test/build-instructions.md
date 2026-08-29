# Build Instructions — price-checker

## Prerequisites
- **Build tool**: Go **1.23+** (module-aware). No `make` required.
- **Dependencies**: fetched by `go` from the module proxy — `aws-sdk-go-v2` (+ `config`, `service/pricing`), `urfave/cli/v2`, `golang.org/x/sync`, `gopkg.in/yaml.v3`, `santhosh-tekuri/jsonschema/v5` (test), `pgregory.net/rapid` (test).
- **Environment variables**: none to build. To *run* against AWS: standard AWS credentials (env, `~/.aws`, `--profile`, SSO, or instance role) with `pricing:GetProducts`.
- **System**: any OS Go supports; ~200 MB disk for the module cache; no cgo.
- **Optional runtime dep**: the `terraform` binary, only for `--path <directory>` mode.

## Build steps

### 1. Fetch and pin dependencies
```bash
cd price-checker
go mod tidy          # REQUIRED once: writes go.sum, resolves transitive AWS SDK modules
go mod download
```
> The versions in `go.mod` are indicative. `go mod tidy` will adjust patch versions and add the transitive `aws-sdk-go-v2/credentials`, `smithy-go`, etc. Commit the resulting `go.mod` + `go.sum`.

### 2. Format & vet
```bash
gofmt -l .           # must print nothing
go vet ./...
```

### 3. Build the binary
```bash
go build -o bin/price-checker ./cmd/price-checker
./bin/price-checker version
```

### 4. Cross-compile the release targets
```bash
for t in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
  GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 \
  go build -trimpath -ldflags "-s -w" -o /tmp/pc-${t%/*}-${t#*/} ./cmd/price-checker
done
```

### 5. Verify
- **Expected output**: `go build` exits 0; `price-checker version` prints `price-checker dev (commit none, built unknown, go1.23.x)`.
- **Artifacts**: `bin/price-checker` (~15–25 MB static binary).
- **Acceptable warnings**: none. `golangci-lint` may flag style nits; the CI config treats lint failures as errors — fix or `//nolint` with a reason.

## Troubleshooting

### `go mod tidy` cannot resolve a module
- **Cause**: offline, or a pinned version was yanked.
- **Fix**: ensure network access to `proxy.golang.org`; if a version is gone, run `go get <module>@latest` then `go mod tidy`.

### Compilation error in `internal/cli` about `urfave/cli/v2` symbols
- **Cause**: a major-version skew in the CLI library API.
- **Fix**: this code targets `urfave/cli/v2` (v2.27.x). Pin it: `go get github.com/urfave/cli/v2@v2.27.4`.

### `//go:embed` error: "pattern html/report.css: no matching files"
- **Cause**: building from outside the module root or a partial checkout.
- **Fix**: build from the repo root; ensure `internal/render/html/` contains `report.html.tmpl`, `report.css`, `report.js`.

### AWS calls fail with `UnrecognizedClientException` / `AccessDenied`
- **Cause**: missing or wrong credentials (a runtime, not build, issue).
- **Fix**: configure credentials; the tool prints the actionable message. `price-checker version` and `usage generate` work with no credentials.
