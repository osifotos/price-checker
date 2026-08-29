# U1 `plan-ingest` — Code Summary

**Location**: `internal/plan/` · **Imports**: stdlib only (verified by `imports_test.go`)

> Go toolchain not present in this environment — `go build`/`go test`/`gofmt` run in Build and Test.

## Files created

| File | Contents |
|---|---|
| `doc.go` | package doc |
| `model.go` | `Plan`, `Resource`, `ChangeAction` (+6 consts), `ProviderConfig`; accessors `Resources()`, `PriorResources()`, `DeletedResources()`, `AllResources()` — all sorted by address |
| `attrmap.go` | `AttrMap` read-only view over `after` + `after_unknown`: `String/Int/Float/Bool` (return known-bool), `IsKnown`, `Has`, `Block`, `Blocks`, `StringSlice`, `Keys` |
| `wire.go` | private `planWire`/`resourceChangeWire`/`configWire` mirrors of `terraform show -json`; `decodeObject` helper |
| `parser.go` | `PlanParser` + `NewParser()`; `format_version` validation (0.1–0.2, 1.x), action mapping, provider-config + provider-key-by-address walk, `stripIndices` for count/for_each addresses |
| `loader.go` | `PlanLoader` + `NewLoader()`; stdin / file / directory modes; `runner` interface (`execRunner` default) so tests never shell out; temp plan file cleanup; actionable "terraform not found" error |
| `testdata/ec2_s3_natgw.json` | plan fixture: count-expanded EC2, module NAT gateway, S3, pure-delete EIP, data source, replace RDS, two provider configs (aliased) |

## Tests

| File | Covers |
|---|---|
| `parser_test.go` | fixture parse, action mapping (create/delete/read/replace), module address, provider-config key (incl. index-stripped + aliased), prior resources, `format_version` acceptance/rejection matrix, malformed JSON, empty-plan-is-valid, determinism |
| `attrmap_test.go` | known-after-apply (`arn` unknown), nested `Block` access + nested unknown, numeric/bool getters, prior attributes, missing keys |
| `loader_test.go` | stdin (+empty), file (+missing), directory with fake `runner`: terraform-not-found error, success path issues `plan` then `show` |
| `pbt_test.go` | **PBT-03/07/08**: generated plans preserve the managed-resource address set and each attribute's known/unknown classification (`rapid`) |
| `imports_test.go` | **NFR-5.3** no third-party imports in non-test files |

## Stories advanced

- **US-LOCAL-ESTIMATE-1** (owned): plan JSON from file or stdin → model; `format_version` 1.x accepted, others rejected with a clear message; malformed JSON reported; delete-only resources excluded from forward cost; deterministic ordering.
- **US-LOCAL-ESTIMATE-2** (owned): directory input runs `terraform plan`/`show -json`; missing `terraform` → actionable error naming the file/stdin alternative; temp file cleaned up.
- Supports US-PRICING-3 (`ProviderConfigs` + `ProviderConfigKey` for U2 RegionResolver), US-PRICING-4 (`AttrMap` for pricers), US-DIFF-1 (`PriorResources()`).

## Notes / deviations

- `planned_values` is intentionally ignored; `resource_changes` is the sole source (only it carries `after_unknown`).
- Deep dotted attribute paths not implemented; `Block`/`Blocks` cover every catalogued resource (one nesting level).
- `configuration` block may be absent in some `terraform show` outputs → `ProviderConfigs` empty, region falls to `--aws-region` in U2.
