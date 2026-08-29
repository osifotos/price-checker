# U1 `plan-ingest` — Code Generation Plan (consolidated cadence)

Code location: `internal/plan/`. Stories: US-LOCAL-ESTIMATE-1, US-LOCAL-ESTIMATE-2.

| # | Step | Status |
|---|---|---|
| 1 | `doc.go`, `model.go` (Plan/Resource/ChangeAction/ProviderConfig + accessors) | [x] |
| 2 | `attrmap.go` (AttrMap + getters + unknown propagation) | [x] |
| 3 | `wire.go` (private Terraform JSON mirrors) | [x] |
| 4 | `parser.go` (Parse, format_version validation, action mapping, provider config) | [x] |
| 5 | `loader.go` (stdin/file/dir modes, runner interface) | [x] |
| 6 | `testdata/ec2_s3_natgw.json` fixture | [x] |
| 7 | Tests: parser, attrmap, loader (fake runner), PBT, import-lint | [x] |
| 8 | `code-summary.md` | [x] |

Build/lint/test deferred to Build and Test stage.
