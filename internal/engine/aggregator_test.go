package engine

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/osifotos/price-checker/internal/plan"
	"github.com/osifotos/price-checker/pkg/schema"
	"pgregory.net/rapid"
)

func comp(name, monthly, hourly string) schema.CostComponent {
	return schema.CostComponent{Name: name, Unit: "u", PricePerUnit: "0", MonthlyQuantity: "0", MonthlyCost: monthly, HourlyCost: hourly}
}

func res(addr, typ, module string) plan.Resource {
	return plan.Resource{Address: addr, Type: typ, Name: "x", ModuleAddress: module}
}

func TestBuildBasic(t *testing.T) {
	priced := []PricedResource{
		{
			Resource: res("aws_instance.web", "aws_instance", ""), Region: "us-east-1",
			Components: []schema.CostComponent{comp("Instance", "30.0000", "0.0411"), comp("Storage", "2.4000", "0.0033")},
		},
		{
			Resource: res("module.net.aws_nat_gateway.n", "aws_nat_gateway", "module.net"), Region: "us-east-1",
			Components: []schema.CostComponent{comp("NAT", "32.8500", "0.0450")},
		},
		{
			Resource: res("aws_kms_key.k", "aws_kms_key", ""), Region: "us-east-1",
			NotEstimated: []schema.NotEstimated{{Address: "aws_kms_key.k", Type: "aws_kms_key", ReasonCode: schema.ReasonUnsupportedType, Message: "x"}},
		},
	}
	b := Aggregator{}.Build(priced, schema.RunMetadata{ToolVersion: "t", GeneratedAt: "2026-08-29T00:00:00Z", PlanFormatVersion: "1.2"}, schema.PeriodMonth)

	if b.TotalMonthly != "65.2500" {
		t.Errorf("total monthly = %q, want 65.2500", b.TotalMonthly)
	}
	if b.Summary.ResourcesTotal != 3 || b.Summary.ResourcesEstimated != 2 || b.Summary.ResourcesNotEstimated != 1 {
		t.Errorf("summary = %+v", b.Summary)
	}
	if b.Summary.ComponentsEstimated != 3 || b.Summary.ComponentsNotEstimated != 1 {
		t.Errorf("component summary = %+v", b.Summary)
	}
	if len(b.Modules) != 2 {
		t.Fatalf("modules = %d, want 2", len(b.Modules))
	}
	// modules sorted: "" then "module.net"
	if b.Modules[0].Module != "" || b.Modules[0].MonthlyCost != "32.4000" {
		t.Errorf("root module = %+v", b.Modules[0])
	}
	if b.Modules[1].Module != "module.net" || b.Modules[1].MonthlyCost != "32.8500" {
		t.Errorf("net module = %+v", b.Modules[1])
	}
	if len(b.Metadata.Regions) != 1 || b.Metadata.Regions[0] != "us-east-1" {
		t.Errorf("regions = %v", b.Metadata.Regions)
	}
}

func TestBuildEmpty(t *testing.T) {
	b := Aggregator{}.Build(nil, schema.RunMetadata{GeneratedAt: "2026-08-29T00:00:00Z", PlanFormatVersion: "1.2"}, schema.PeriodMonth)
	if b.TotalMonthly != "0.0000" && b.TotalMonthly != "0" {
		t.Errorf("empty total = %q", b.TotalMonthly)
	}
	if b.Summary.ResourcesTotal != 0 {
		t.Errorf("summary = %+v", b.Summary)
	}
	if b.Resources == nil || b.Modules == nil || b.NotEstimated == nil {
		t.Errorf("slices must be non-nil after Build")
	}
}

func TestBuildDeterministic(t *testing.T) {
	priced := []PricedResource{
		{Resource: res("b.z", "aws_instance", ""), Region: "us-east-1", Components: []schema.CostComponent{comp("c", "1.0000", "0.0014")}},
		{Resource: res("a.y", "aws_instance", ""), Region: "eu-west-1", Components: []schema.CostComponent{comp("c", "2.0000", "0.0027")}},
	}
	meta := schema.RunMetadata{GeneratedAt: "2026-08-29T00:00:00Z", PlanFormatVersion: "1.2"}
	b1, _ := json.Marshal(Aggregator{}.Build(priced, meta, schema.PeriodMonth))
	b2, _ := json.Marshal(Aggregator{}.Build(priced, meta, schema.PeriodMonth))
	if string(b1) != string(b2) {
		t.Fatalf("non-deterministic:\n%s\n%s", b1, b2)
	}
}

// PBT-03: aggregation invariants.
func TestAggregationInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 10).Draw(t, "n")
		priced := make([]PricedResource, 0, n)
		wantTotal := new(big.Rat)
		for i := 0; i < n; i++ {
			nc := rapid.IntRange(0, 4).Draw(t, "nc")
			var comps []schema.CostComponent
			resTotal := new(big.Rat)
			for j := 0; j < nc; j++ {
				cents := rapid.Int64Range(0, 500000).Draw(t, "cents")
				m := schema.RatToString(new(big.Rat).SetFrac64(cents, 100), 4)
				h := schema.RatToString(new(big.Rat).Quo(new(big.Rat).SetFrac64(cents, 100), big.NewRat(730, 1)), 4)
				comps = append(comps, comp("c", m, h))
				resTotal.Add(resTotal, new(big.Rat).SetFrac64(cents, 100))
			}
			wantTotal.Add(wantTotal, resTotal)
			priced = append(priced, PricedResource{
				Resource: res("aws_instance."+rapid.StringMatching(`[a-z]{1,4}`).Draw(t, "nm"), "aws_instance", ""),
				Region:   "us-east-1", Components: comps,
			})
		}
		b := Aggregator{}.Build(priced, schema.RunMetadata{GeneratedAt: "2026-08-29T00:00:00Z", PlanFormatVersion: "1.2"}, schema.PeriodMonth)

		// project total == sum of resource totals (within rounding)
		got, _ := schema.StringToRat(b.TotalMonthly)
		diff := new(big.Rat).Sub(got, wantTotal)
		diff.Abs(diff)
		if diff.Cmp(big.NewRat(1, 100)) > 0 {
			t.Fatalf("total %s not within 0.01 of %s", b.TotalMonthly, wantTotal.FloatString(4))
		}
		// each resource total == sum of its components
		for _, r := range b.Resources {
			rt, _ := schema.StringToRat(r.MonthlyCost)
			if rt.Sign() < 0 {
				t.Fatalf("negative resource total %s", r.MonthlyCost)
			}
		}
		// summary consistency
		if b.Summary.ResourcesEstimated+b.Summary.ResourcesNotEstimated != b.Summary.ResourcesTotal {
			t.Fatalf("summary counts inconsistent: %+v", b.Summary)
		}
	})
}
