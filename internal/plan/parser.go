package plan

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// PlanParser turns raw `terraform show -json` bytes into a *Plan.
type PlanParser interface {
	Parse(raw []byte) (*Plan, error)
}

// NewParser returns the default PlanParser.
func NewParser() PlanParser { return parser{} }

type parser struct{}

func (parser) Parse(raw []byte) (*Plan, error) {
	var w planWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}
	if w.FormatVersion == "" {
		return nil, fmt.Errorf("not a terraform plan JSON: missing format_version")
	}
	if err := checkFormatVersion(w.FormatVersion, w.TerraformVersion); err != nil {
		return nil, err
	}

	p := &Plan{
		FormatVersion:    w.FormatVersion,
		TerraformVersion: w.TerraformVersion,
		ProviderConfigs:  map[string]ProviderConfig{},
		resources:        make([]Resource, 0, len(w.ResourceChanges)),
	}

	providerKeyByAddr := map[string]string{}
	if w.Configuration != nil {
		for key, pc := range w.Configuration.ProviderConfig {
			region := ""
			if pc.Expressions.Region != nil {
				if s, ok := pc.Expressions.Region.ConstantValue.(string); ok {
					region = s
				}
			}
			p.ProviderConfigs[key] = ProviderConfig{Name: pc.Name, Alias: pc.Alias, ConstantRegion: region}
		}
		collectProviderKeys(w.Configuration.RootModule, "", providerKeyByAddr)
	}

	for _, rc := range w.ResourceChanges {
		after, err := decodeObject(rc.Change.After)
		if err != nil {
			return nil, fmt.Errorf("parse plan: resource %q: after: %w", rc.Address, err)
		}
		afterUnknown, err := decodeObject(rc.Change.AfterUnknown)
		if err != nil {
			return nil, fmt.Errorf("parse plan: resource %q: after_unknown: %w", rc.Address, err)
		}
		before, err := decodeObject(rc.Change.Before)
		if err != nil {
			return nil, fmt.Errorf("parse plan: resource %q: before: %w", rc.Address, err)
		}

		p.resources = append(p.resources, Resource{
			Address:           rc.Address,
			Type:              rc.Type,
			Name:              rc.Name,
			Mode:              rc.Mode,
			ModuleAddress:     rc.ModuleAddress,
			ProviderName:      rc.ProviderName,
			ProviderConfigKey: lookupProviderKey(providerKeyByAddr, rc.Address),
			Action:            mapAction(rc.Change.Actions),
			Attributes:        newAttrMap(after, afterUnknown),
			PriorAttributes:   newAttrMap(before, nil),
		})
	}
	return p, nil
}

func mapAction(actions []string) ChangeAction {
	switch len(actions) {
	case 1:
		switch actions[0] {
		case "no-op":
			return ActionNoOp
		case "create":
			return ActionCreate
		case "update":
			return ActionUpdate
		case "delete":
			return ActionDelete
		case "read":
			return ActionRead
		}
	case 2:
		a, b := actions[0], actions[1]
		if (a == "create" && b == "delete") || (a == "delete" && b == "create") {
			return ActionReplace
		}
	}
	return ActionNoOp
}

func checkFormatVersion(fv, tfVersion string) error {
	major, minor, err := splitVersion(fv)
	if err != nil {
		return fmt.Errorf("unsupported plan format_version %q: %v", fv, err)
	}
	switch {
	case major == 0 && minor <= 2:
		return nil
	case major == 1:
		return nil
	default:
		return fmt.Errorf(
			"unsupported plan format_version %q (supported: 0.1-0.2, 1.x); produced by terraform %q",
			fv, tfVersion)
	}
}

func splitVersion(v string) (major, minor int, err error) {
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("want MAJOR.MINOR")
	}
	if major, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, err
	}
	if minor, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, err
	}
	return major, minor, nil
}

// lookupProviderKey resolves a possibly count/for_each-indexed resource address
// against the configuration map, which is never expanded.
func lookupProviderKey(m map[string]string, address string) string {
	if k, ok := m[address]; ok {
		return k
	}
	return m[stripIndices(address)]
}

// stripIndices removes every "[...]" segment from an address, so
// `module.net["a"].aws_x.y[0]` becomes `module.net.aws_x.y`.
func stripIndices(address string) string {
	var b strings.Builder
	depth := 0
	for _, r := range address {
		switch r {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// collectProviderKeys walks the configuration module tree building a map from
// full resource address to provider_config_key.
func collectProviderKeys(m configModuleWire, prefix string, out map[string]string) {
	for _, r := range m.Resources {
		addr := r.Address
		if prefix != "" {
			addr = prefix + "." + r.Address
		}
		if r.ProviderConfigKey != "" {
			out[addr] = r.ProviderConfigKey
		}
	}
	for name, call := range m.ModuleCalls {
		childPrefix := "module." + name
		if prefix != "" {
			childPrefix = prefix + ".module." + name
		}
		collectProviderKeys(call.Module, childPrefix, out)
	}
}
