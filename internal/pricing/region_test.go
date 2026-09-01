package pricing

import (
	"testing"

	"github.com/mtosin123/tf-price_checker/internal/plan"
)

func TestRegionResolver(t *testing.T) {
	cfgs := map[string]plan.ProviderConfig{
		"aws":       {Name: "aws", ConstantRegion: "us-east-1"},
		"aws.west":  {Name: "aws", Alias: "west", ConstantRegion: "us-west-2"},
		"aws.novar": {Name: "aws"},
	}
	var rr RegionResolver

	if got, ok := rr.Resolve(plan.Resource{ProviderConfigKey: "aws"}, cfgs, ""); !ok || got != "us-east-1" {
		t.Errorf("provider constant: got %q %v", got, ok)
	}
	if got, ok := rr.Resolve(plan.Resource{ProviderConfigKey: "aws"}, cfgs, "eu-west-1"); !ok || got != "eu-west-1" {
		t.Errorf("override should win: got %q %v", got, ok)
	}
	if got, ok := rr.Resolve(plan.Resource{ProviderConfigKey: "aws.west"}, cfgs, ""); !ok || got != "us-west-2" {
		t.Errorf("aliased provider: got %q %v", got, ok)
	}
	if _, ok := rr.Resolve(plan.Resource{ProviderConfigKey: "aws.novar"}, cfgs, ""); ok {
		t.Error("no constant region and no override should be unresolved")
	}
	if _, ok := rr.Resolve(plan.Resource{ProviderConfigKey: "missing"}, cfgs, ""); ok {
		t.Error("missing provider key should be unresolved")
	}
}
