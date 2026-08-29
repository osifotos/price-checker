package pricing

import "github.com/example/price-checker/internal/plan"

// RegionResolver decides which AWS region a resource is priced in.
type RegionResolver struct{}

// Resolve returns the region for res. Order of precedence:
//
//  1. override (from --aws-region), if non-empty
//  2. the constant region on the resource's provider configuration
//  3. ("", false) — the caller records MISSING_ATTRIBUTE
func (RegionResolver) Resolve(res plan.Resource, cfgs map[string]plan.ProviderConfig, override string) (string, bool) {
	if override != "" {
		return override, true
	}
	if cfg, ok := cfgs[res.ProviderConfigKey]; ok && cfg.ConstantRegion != "" {
		return cfg.ConstantRegion, true
	}
	return "", false
}
