# U1 `plan-ingest` — Consolidated Design

**Unit**: U1 `plan-ingest` · **Package**: `internal/plan` · **Depends on**: U0 `pkg/schema` (only for `schema` types it re-exports? no — U1 is Terraform-only; it depends on stdlib + U0 not required at all in practice, but may reference `schema.ReasonCode` names in docs). Actual imports: stdlib only.
**Owns stories**: US-LOCAL-ESTIMATE-1, US-LOCAL-ESTIMATE-2 · **Supports**: US-PRICING-3, US-PRICING-4, US-DIFF-1

---

## 1. Functional Design

### 1.1 Responsibility

Convert a Terraform plan into a provider-neutral `Plan` model. Two entry points:

1. **`PlanLoader.Load`** — resolve a `--path` value (or stdin) to raw plan JSON bytes.
2. **`PlanParser.Parse`** — parse those bytes into `*Plan`.

Nothing downstream of U1 imports Terraform JSON shapes.

### 1.2 Input modes (`PlanLoader`)

| `source` | Behaviour |
|---|---|
| `""` or `"-"` | read all of `stdin` |
| path to a regular file | read the file |
| path to a directory | run `terraform show -json` in that directory (auto-generating a plan file: `terraform init -input=false` is **not** run — the dir is assumed initialised; if `terraform plan` is needed, run `terraform plan -out` then `terraform show -json <planfile>`). Capture stdout. Remove any temp plan file. |
| path that does not exist | error |

- Directory mode requires the `terraform` binary on `PATH`; absence → error naming the file/stdin alternative.
- `terraform` invocation: `terraform -chdir=<dir> plan -out=<tmp> -input=false` then `terraform -chdir=<dir> show -json <tmp>`. Non-zero exit → error with stderr attached. `tmp` created via `os.CreateTemp`, deleted in a `defer`.
- A context is threaded so a slow `terraform` can be cancelled.
- `LoadInfo{Mode, TerraformVersion}` returned (TerraformVersion filled after parse, actually — Loader returns Mode; Parser fills version. Keep `LoadInfo` = `{Mode string}` from Loader).

### 1.3 Parsing (`PlanParser`)

**Source of truth**: `resource_changes[]`, not `planned_values` — because only `resource_changes` carries `after_unknown` (known-after-apply information).

For each `resource_changes` entry:

| Plan model field | From |
|---|---|
| `Address` | `.address` |
| `Type`, `Name`, `Mode` | `.type`, `.name`, `.mode` |
| `ModuleAddress` | `.module_address` (`""` = root) |
| `ProviderName` | `.provider_name` |
| `Action` | derived from `.change.actions` (see 1.4) |
| `Attributes` | `AttrMap` built from `.change.after` (values) + `.change.after_unknown` (unknown tree) |
| `PriorAttributes` | `AttrMap` from `.change.before` (no unknown tree; prior state is fully known) |

- `Plan.Resources()` returns entries whose action contributes to the **planned** state: `create`, `update`, `no-op`, `create-then-delete`/`delete-then-create` (replace). `delete` entries are excluded from `Resources()` (zero forward cost) but available via `Plan.DeletedResources()`.
- `Plan.PriorResources()` returns entries with a non-null `before` (for single-plan diff derivation): actions `update`, `delete`, `no-op`, replace.
- `count`/`for_each` are already expanded by Terraform; each element is its own `resource_changes` entry with an indexed address.

### 1.4 Change action mapping

| `change.actions` | `Action` |
|---|---|
| `["no-op"]` | `no-op` |
| `["create"]` | `create` |
| `["update"]` | `update` |
| `["delete"]` | `delete` |
| `["create","delete"]` or `["delete","create"]` | `replace` |
| `["read"]` | `read` (data sources) |

### 1.5 `AttrMap`

Wraps `after` (or `before`) values plus the parallel `after_unknown` tree.

| Method | Purpose |
|---|---|
| `String(key) (string, bool)` | top-level string; `bool` is **known** |
| `Int(key) (int64, bool)` / `Float(key) (float64, bool)` | numeric (JSON numbers) |
| `Bool(key) (bool, bool)` | boolean |
| `IsKnown(key) bool` | false if `after_unknown[key] == true` (or a nested unknown makes the whole value unknown) |
| `Has(key) bool` | key present in `after` |
| `Block(key) (AttrMap, bool)` | first element of a nested block list (e.g. `root_block_device`); `bool` = present & is an object |
| `Blocks(key) []AttrMap` | all elements of a nested block list |
| `StringSlice(key) ([]string, bool)` | list of strings (e.g. `security_groups`) |
| `Keys() []string` | present keys, sorted |

- Deep dotted paths are **not** supported in v1 — pricers use `Block`/`Blocks` to descend one level, which covers every catalogued resource.
- Unknown propagation: for a block, if `after_unknown[key]` is `true` the whole block is unknown; if it is a list of objects, per-element unknown maps are attached.

### 1.6 Provider configuration

`configuration.provider_config` is parsed into `Plan.ProviderConfigs map[string]ProviderConfig`:

```
ProviderConfig{ Name, Alias, ConstantRegion string }
```

`ConstantRegion` = `expressions.region.constant_value` when present and a string, else `""`.

`configuration.root_module` (+ nested `module_calls`) is walked to build `address -> provider_config_key`, stored on each `Resource.ProviderConfigKey`. U2's `RegionResolver` maps `ProviderConfigKey -> ProviderConfigs[key].ConstantRegion`, with `--aws-region` override.

If `configuration` is absent (some `terraform show` outputs omit it for plan files without config) → `ProviderConfigs` empty, `ProviderConfigKey` `""`; RegionResolver then relies on `--aws-region` or records `MISSING_ATTRIBUTE`.

### 1.7 Format-version validation

- Accept `format_version` `"0.1"`, `"0.2"`, `"1.0"`, `"1.1"`, `"1.2"` (Terraform 0.12 → 1.9+). Parse `major.minor`; accept `major == 0 && minor <= 2` or `major == 1`.
- Unknown/newer major → error: `unsupported plan format_version %q (supported: 0.1-0.2, 1.x); produced by terraform_version %q`.
- Empty/missing `format_version` → error (`not a terraform plan JSON: missing format_version`).
- Malformed JSON → error with byte offset (`encoding/json` `SyntaxError`).

### 1.8 Error scenarios

| Case | Result |
|---|---|
| stdin empty | `error: empty plan input` |
| file not found | `error: open <path>: no such file` (wrapped) |
| dir mode, no `terraform` | `error: -path is a directory but 'terraform' was not found on PATH; pass a plan JSON file or pipe 'terraform show -json' instead` |
| `terraform` exits non-zero | `error: terraform show -json failed: <stderr>` |
| not JSON | `error: parse plan: invalid character ... at offset N` |
| JSON but not a plan | `error: not a terraform plan JSON: missing format_version` |
| unsupported version | as 1.7 |
| `resource_changes` absent | valid empty `Plan` (0 resources) — some plans genuinely have none |

### 1.9 Edge cases

- Nested modules `module.a.module.b.aws_x.y` → `ModuleAddress = "module.a.module.b"`.
- Indexed addresses `aws_instance.web[0]`, `aws_instance.web["blue"]` → preserved verbatim in `Address`; `Name` = `web`.
- `after` is `null` (pure delete) → `Attributes` is an empty `AttrMap`, `Action = delete`.
- Sensitive values: `after` still contains them; U1 does not scrub (U3/U4 only surface cost-relevant fields). `after_sensitive` is ignored.
- Very large plan (10k entries) → single pass, pre-sized slice from `len(resource_changes)`.
- Duplicate addresses (shouldn't happen) → last wins, warning not emitted (parser is silent; Estimator may log).

---

## 2. NFR Requirements (unit-scoped)

| NFR | Obligation |
|---|---|
| NFR-2.1 | Deterministic: `Resources()` sorted by address |
| NFR-3.2 | No network except the explicit `terraform` subprocess in dir mode; no telemetry |
| NFR-4.1/4.2 | Actionable errors for every failure mode in 1.8 |
| NFR-5.3 | `internal/plan` imports **stdlib only** (`encoding/json`, `os`, `os/exec`, `context`, `sort`, `strings`, `fmt`, `path/filepath`) |
| NFR-6 | Pure Go, no cgo |
| NFR-2.3 (PBT) | PBT-03 invariant: parsing then reading back preserves address set and known/unknown classification; PBT-07 generator produces synthetic `resource_changes` documents |

### 2.1 Dependencies

Stdlib only. No third-party imports.

---

## 3. NFR Design

- **`os/exec` isolation**: all `terraform` invocation is behind an unexported `runner` interface (`run(ctx, dir string, args ...string) (stdout, stderr []byte, err error)`) so tests inject a fake and never shell out. Default impl uses `exec.CommandContext`.
- **Streaming**: `Load` returns `[]byte` (plans are typically < 10 MB); no need to stream. `Parse` uses `json.Unmarshal` into intermediate wire structs (`planWire`, `resourceChangeWire`) then maps to the model.
- **Wire structs kept private**; the public model is hand-mapped so schema drift in Terraform's format is contained to `wire.go`.
- **`AttrMap` immutability**: constructed once, getters are read-only, no setters.

### 3.1 File plan (`internal/plan/`)

| File | Contents |
|---|---|
| `doc.go` | package doc |
| `model.go` | `Plan`, `Resource`, `ChangeAction` + consts, `ProviderConfig`, accessors (`Resources`, `PriorResources`, `DeletedResources`, `RegionHintFor`) |
| `attrmap.go` | `AttrMap` + all getters + unknown propagation |
| `loader.go` | `PlanLoader` interface + `fileLoader`; `runner` interface + `execRunner` |
| `parser.go` | `PlanParser` interface + `parser`; `format_version` validation; action mapping |
| `wire.go` | private `planWire`, `resourceChangeWire`, `configWire`, unmarshal helpers |
| `testdata/*.json` | plan fixtures |
| `*_test.go` | example + PBT tests |

### 3.2 Test targets (DoD)

- `TestParseMinimalPlan`, `TestParseNestedModules`, `TestParseCountExpanded`, `TestParseReplaceAction`, `TestParseDeleteOnly`, `TestParseDataSource`.
- `TestKnownAfterApply` — `after_unknown` correctly makes `IsKnown` false.
- `TestUnsupportedFormatVersion`, `TestNotAPlan`, `TestMalformedJSON`.
- `TestProviderConfigRegion` — constant region extracted; aliased provider; missing configuration.
- `TestLoaderStdin`, `TestLoaderFile`, `TestLoaderDirNoTerraform` (fake runner), `TestLoaderDirSuccess` (fake runner returns fixture bytes).
- `TestResourcesSortedByAddress` (determinism).
- **PBT** (`rapid`): `TestParseRoundTripPreservesAddresses` — generate a synthetic plan wire doc, marshal to JSON, parse, assert address set + per-attr known/unknown match the generator's intent (PBT-03/07/08).
- Recorded fixture: a real `terraform show -json` sample committed under `testdata/` (`ec2_s3_natgw.json`).

---

## 4. Open items → later units

- `RegionResolver` (consumes `ProviderConfigs` + `ProviderConfigKey`) lives in U2.
- Whether pricers need deep-path attribute access → revisit if any catalogued resource needs > 1 level of nesting (none identified).
