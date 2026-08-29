package estimator

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/example/price-checker/internal/engine"
	"github.com/example/price-checker/internal/plan"
	"github.com/example/price-checker/internal/pricing"
	"github.com/example/price-checker/pkg/schema"
)

// ---- fakes ----

type fakeLoader struct{ raw string }

func (f fakeLoader) Load(ctx context.Context, source string, stdin io.Reader) ([]byte, plan.LoadInfo, error) {
	return []byte(f.raw), plan.LoadInfo{Mode: "stdin"}, nil
}

type fakePricer struct {
	types      []string
	components []schema.CostComponent
	err        error
}

func (p fakePricer) ResourceTypes() []string { return p.types }
func (p fakePricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	if p.err != nil {
		return pricing.PricingOutput{}, p.err
	}
	return pricing.PricingOutput{Components: p.components}, nil
}

type fakeQuerier struct{ stats pricing.QueryStats }

func (q fakeQuerier) Query(ctx context.Context, _ pricing.PriceQuery) ([]pricing.PriceDimension, error) {
	return nil, nil
}
func (q fakeQuerier) Stats() pricing.QueryStats { return q.stats }

const planWithRegion = `{
  "format_version": "1.3",
  "terraform_version": "1.9.0",
  "resource_changes": [
    { "address": "aws_instance.web", "type": "aws_instance", "name": "web", "mode": "managed", "module_address": "",
      "change": { "actions": ["create"], "after": { "instance_type": "t3.medium" }, "after_unknown": {} } },
    { "address": "aws_kms_key.k", "type": "aws_kms_key", "name": "k", "mode": "managed", "module_address": "",
      "change": { "actions": ["create"], "after": {}, "after_unknown": {} } }
  ],
  "configuration": {
    "provider_config": { "aws": { "name": "aws", "expressions": { "region": { "constant_value": "us-east-1" } } } },
    "root_module": { "resources": [
      { "address": "aws_instance.web", "provider_config_key": "aws" },
      { "address": "aws_kms_key.k", "provider_config_key": "aws" }
    ] }
  }
}`

func newEstimator(t *testing.T, raw string, pricers ...pricing.Pricer) *Estimator {
	t.Helper()
	return New(Deps{
		Loader:      fakeLoader{raw: raw},
		Parser:      plan.NewParser(),
		Catalog:     pricing.NewCatalog(pricers...),
		Querier:     fakeQuerier{stats: pricing.QueryStats{Issued: 3, CacheHits: 1, CacheMisses: 3, DedupHits: 2}},
		Region:      pricing.RegionResolver{},
		Agg:         engine.Aggregator{},
		ToolVersion: "9.9.9",
		Now:         func() time.Time { return time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC) },
	})
}

func TestEstimateHappyAndUnsupported(t *testing.T) {
	e := newEstimator(t, planWithRegion, fakePricer{
		types:      []string{"aws_instance"},
		components: []schema.CostComponent{{Name: "Instance", Unit: "hours", PricePerUnit: "0.0416", MonthlyQuantity: "730", MonthlyCost: "30.3680", HourlyCost: "0.0416"}},
	})
	b, err := e.Estimate(context.Background(), EstimateRequest{Period: schema.PeriodMonth})
	if err != nil {
		t.Fatal(err)
	}
	if b.TotalMonthly != "30.3680" {
		t.Errorf("total = %q, want 30.3680", b.TotalMonthly)
	}
	if b.Summary.ResourcesTotal != 2 || b.Summary.ResourcesEstimated != 1 {
		t.Errorf("summary = %+v", b.Summary)
	}
	var kmsReason schema.ReasonCode
	for _, ne := range b.NotEstimated {
		if ne.Address == "aws_kms_key.k" {
			kmsReason = ne.ReasonCode
		}
	}
	if kmsReason != schema.ReasonUnsupportedType {
		t.Errorf("kms reason = %q, want UNSUPPORTED_TYPE", kmsReason)
	}
	if b.Metadata.GeneratedAt != "2026-08-29T00:00:00Z" {
		t.Errorf("GeneratedAt = %q", b.Metadata.GeneratedAt)
	}
	if b.Metadata.ToolVersion != "9.9.9" || b.Metadata.PriceQueries != 3 || b.Metadata.DedupHits != 2 {
		t.Errorf("metadata = %+v", b.Metadata)
	}
	if b.Metadata.PlanFormatVersion != "1.3" {
		t.Errorf("plan format = %q", b.Metadata.PlanFormatVersion)
	}
}

func TestEstimateMissingRegion(t *testing.T) {
	planNoConfig := `{"format_version":"1.2","resource_changes":[
	  {"address":"aws_instance.web","type":"aws_instance","name":"web","mode":"managed","module_address":"",
	   "change":{"actions":["create"],"after":{"instance_type":"t3.medium"},"after_unknown":{}}}]}`
	e := newEstimator(t, planNoConfig, fakePricer{types: []string{"aws_instance"}})
	b, err := e.Estimate(context.Background(), EstimateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(b.NotEstimated) != 1 || b.NotEstimated[0].ReasonCode != schema.ReasonMissingAttribute {
		t.Errorf("want MISSING_ATTRIBUTE, got %+v", b.NotEstimated)
	}
	// with an override it resolves
	b2, _ := e.Estimate(context.Background(), EstimateRequest{RegionOverride: "eu-west-1"})
	if b2.Summary.ResourcesEstimated != 0 && len(b2.Resources) != 1 {
		t.Errorf("override should let the resource be priced: %+v", b2.Summary)
	}
}

func TestEstimatePricerError(t *testing.T) {
	e := newEstimator(t, planWithRegion, fakePricer{types: []string{"aws_instance"}, err: context.DeadlineExceeded})
	b, err := e.Estimate(context.Background(), EstimateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	got := ""
	for _, ne := range b.NotEstimated {
		if ne.Address == "aws_instance.web" {
			got = string(ne.ReasonCode)
		}
	}
	if got != string(schema.ReasonPricingAPIError) {
		t.Errorf("want PRICING_API_ERROR, got %q", got)
	}
}

func TestEstimateLoadError(t *testing.T) {
	e := newEstimator(t, "not json {", fakePricer{types: []string{"aws_instance"}})
	if _, err := e.Estimate(context.Background(), EstimateRequest{}); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestEstimateDeterministic(t *testing.T) {
	e := newEstimator(t, planWithRegion, fakePricer{
		types:      []string{"aws_instance"},
		components: []schema.CostComponent{{Name: "Instance", Unit: "hours", PricePerUnit: "0.0416", MonthlyQuantity: "730", MonthlyCost: "30.3680", HourlyCost: "0.0416"}},
	})
	var prev string
	for i := 0; i < 5; i++ {
		b, err := e.Estimate(context.Background(), EstimateRequest{})
		if err != nil {
			t.Fatal(err)
		}
		s := b.TotalMonthly + "|" + strings.Join(addresses(b), ",")
		if i > 0 && s != prev {
			t.Fatalf("run %d differs: %q vs %q", i, s, prev)
		}
		prev = s
	}
}

func TestEstimateBothStates(t *testing.T) {
	planReplace := `{"format_version":"1.2","terraform_version":"1.9.0","resource_changes":[
	  {"address":"aws_instance.web","type":"aws_instance","name":"web","mode":"managed","module_address":"",
	   "change":{"actions":["update"],
	     "before":{"instance_type":"t3.small"},
	     "after":{"instance_type":"t3.large"},"after_unknown":{}}}],
	  "configuration":{"provider_config":{"aws":{"name":"aws","expressions":{"region":{"constant_value":"us-east-1"}}}},
	    "root_module":{"resources":[{"address":"aws_instance.web","provider_config_key":"aws"}]}}}`

	// pricer returns a cost that depends on instance_type
	sized := sizedPricer{}
	e := newEstimator(t, planReplace, sized)
	prior, planned, err := e.EstimateBothStates(context.Background(), EstimateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if prior.TotalMonthly == planned.TotalMonthly {
		t.Fatalf("prior (%s) and planned (%s) should differ", prior.TotalMonthly, planned.TotalMonthly)
	}
}

type sizedPricer struct{}

func (sizedPricer) ResourceTypes() []string { return []string{"aws_instance"} }
func (sizedPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	it, _ := in.Resource.Attributes.String("instance_type")
	price := "10.0000"
	if it == "t3.large" {
		price = "40.0000"
	}
	return pricing.PricingOutput{Components: []schema.CostComponent{
		{Name: "Instance", Unit: "hours", PricePerUnit: "0", MonthlyQuantity: "1", MonthlyCost: price, HourlyCost: "0.0100"},
	}}, nil
}

func addresses(b *schema.Breakdown) []string {
	out := make([]string, len(b.Resources))
	for i, r := range b.Resources {
		out[i] = r.Address
	}
	return out
}
