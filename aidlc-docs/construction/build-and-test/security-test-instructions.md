# Security Test Instructions — price-checker

The Security extension is **disabled** for this project (Requirements Analysis).
These checks cover the baseline privacy/hygiene guarantees the requirements do
make (NFR-3).

## 1. Dependency scanning
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```
- **Pass**: no known vulnerabilities in the resolved module graph. Re-run in CI.

## 2. Static analysis
```bash
go vet ./...
golangci-lint run          # includes gosec via the default set if enabled
```

## 3. No plan data leaves the host (NFR-3.1/3.2)
- **Assert (code review + test)**: only `internal/awsauth` and `internal/pricing` reach the network; `internal/pricing/aws` has no SDK import (`imports_test.go`).
- **Assert**: `pricing.PriceQuery` carries only `ServiceCode`, `RegionCode`, attribute `Filters` — never a resource name or attribute value. Grep the pricers for any `res.Address` / attribute value flowing into a `Filter`.
- **Test**: run with a plan whose resource attributes contain a fake secret; capture outbound requests with a proxy; confirm the secret never appears.

## 4. Cache contains no plan-derived data (NFR-3.2/3.4)
```bash
./bin/price-checker --path plan.json > /dev/null
find ~/.price-checker/cache -type f -exec grep -l "my-secret-bucket-name" {} \;   # must find nothing
find ~/.price-checker/cache -type f ! -perm 600                                    # must be empty
find ~/.price-checker/cache -type d ! -perm 700                                    # must be empty
```
- `internal/pricing/cache_test.go::TestCacheFilePerms` asserts `0600` in unit tests.

## 5. Output contains only cost-relevant fields (NFR-3.3)
- **Assert**: `schema.Breakdown` / `DiffResult` have no free-form attribute field; renderers only emit `pkg/schema` values. HTML report is XSS-safe via `html/template` auto-escaping (`render_test.go::TestHTMLRendererSelfContained`).

## 6. HTML report is self-contained (FR-10.1)
- `render_test.go` asserts no `http://` / `https://` / `src="//"` in output. Manually open a generated report with the network disabled and confirm it renders fully.

## 7. Credential handling
- Credentials are never logged (only `slog` at info/debug logs service/region/purpose, never secrets — review `client.go` log calls).
- `--profile` and the SDK chain are the only credential inputs; no custom credential file parsing.
