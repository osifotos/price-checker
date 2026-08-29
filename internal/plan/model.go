package plan

import "sort"

// ChangeAction is the normalised planned action for a resource.
type ChangeAction string

const (
	ActionNoOp    ChangeAction = "no-op"
	ActionCreate  ChangeAction = "create"
	ActionUpdate  ChangeAction = "update"
	ActionDelete  ChangeAction = "delete"
	ActionReplace ChangeAction = "replace"
	ActionRead    ChangeAction = "read"
)

// Plan is the provider-neutral form of a Terraform plan.
type Plan struct {
	FormatVersion    string
	TerraformVersion string

	// resources holds every managed and data resource from resource_changes,
	// in the order encountered. Use the accessor methods rather than this slice.
	resources []Resource

	// ProviderConfigs maps a provider config key ("aws", "aws.west",
	// "module.net:aws") to its parsed configuration.
	ProviderConfigs map[string]ProviderConfig
}

// Resource is one planned resource.
type Resource struct {
	Address           string
	Type              string
	Name              string
	Mode              string // "managed" or "data"
	ModuleAddress     string // "" for the root module
	ProviderName      string // e.g. "registry.terraform.io/hashicorp/aws"
	ProviderConfigKey string // key into Plan.ProviderConfigs; "" if unknown
	Action            ChangeAction
	Attributes        AttrMap // planned (after) state
	PriorAttributes   AttrMap // prior (before) state; empty for creates
}

// ProviderConfig is the parsed form of one configuration.provider_config entry.
type ProviderConfig struct {
	Name           string
	Alias          string
	ConstantRegion string // expressions.region.constant_value, "" if not constant
}

// Resources returns the resources that contribute to the planned (post-apply)
// state — everything except pure deletes and data-source reads — sorted by
// address.
func (p *Plan) Resources() []Resource {
	out := make([]Resource, 0, len(p.resources))
	for _, r := range p.resources {
		switch r.Action {
		case ActionCreate, ActionUpdate, ActionNoOp, ActionReplace:
			if r.Mode == "managed" {
				out = append(out, r)
			}
		}
	}
	sortByAddress(out)
	return out
}

// PriorResources returns the resources that existed in the prior state — used to
// derive a diff from a single plan — sorted by address.
func (p *Plan) PriorResources() []Resource {
	out := make([]Resource, 0, len(p.resources))
	for _, r := range p.resources {
		switch r.Action {
		case ActionUpdate, ActionDelete, ActionNoOp, ActionReplace:
			if r.Mode == "managed" && !r.PriorAttributes.isEmpty() {
				out = append(out, r)
			}
		}
	}
	sortByAddress(out)
	return out
}

// DeletedResources returns managed resources being destroyed by this plan.
func (p *Plan) DeletedResources() []Resource {
	out := make([]Resource, 0)
	for _, r := range p.resources {
		if r.Action == ActionDelete && r.Mode == "managed" {
			out = append(out, r)
		}
	}
	sortByAddress(out)
	return out
}

// AllResources returns every parsed resource (managed and data, all actions),
// sorted by address. Mainly for diagnostics and the usage-skeleton generator.
func (p *Plan) AllResources() []Resource {
	out := append([]Resource(nil), p.resources...)
	sortByAddress(out)
	return out
}

func sortByAddress(rs []Resource) {
	sort.Slice(rs, func(i, j int) bool { return rs[i].Address < rs[j].Address })
}
