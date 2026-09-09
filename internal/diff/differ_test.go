package diff_test

import (
	"math/big"
	"testing"

	"github.com/osifotos/price-checker/internal/diff"
	"github.com/osifotos/price-checker/pkg/schema"
	"pgregory.net/rapid"
)

func bd(total string, rs ...schema.Resource) *schema.Breakdown {
	b := schema.NewBreakdown(schema.PeriodMonth)
	b.Resources = rs
	b.TotalMonthly = total
	b.TotalHourly = "0"
	b.Metadata = schema.RunMetadata{ToolVersion: "t", GeneratedAt: "2026-08-29T00:00:00Z", PlanFormatVersion: "1.2"}
	b.Canonicalize()
	return b
}

func r(addr, monthly string, comps int) schema.Resource {
	res := schema.Resource{Address: addr, Type: "aws_instance", Name: "x", MonthlyCost: monthly, HourlyCost: "0"}
	for i := 0; i < comps; i++ {
		res.CostComponents = append(res.CostComponents, schema.CostComponent{Name: "c", MonthlyCost: monthly, HourlyCost: "0"})
	}
	return res
}

func TestDiffClassification(t *testing.T) {
	base := bd("40.0000", r("keep", "10.0000", 1), r("change", "20.0000", 1), r("gone", "10.0000", 1))
	prop := bd("95.0000", r("keep", "10.0000", 1), r("change", "50.0000", 1), r("new", "35.0000", 1))

	d := diff.Differ{}.Diff(base, prop)

	kinds := map[string]schema.DeltaKind{}
	for _, c := range d.Changes {
		kinds[c.Address] = c.Kind
	}
	if kinds["keep"] != schema.DeltaUnchanged {
		t.Errorf("keep = %q", kinds["keep"])
	}
	if kinds["change"] != schema.DeltaChanged {
		t.Errorf("change = %q", kinds["change"])
	}
	if kinds["new"] != schema.DeltaAdded {
		t.Errorf("new = %q", kinds["new"])
	}
	if kinds["gone"] != schema.DeltaRemoved {
		t.Errorf("gone = %q", kinds["gone"])
	}

	if d.TotalDeltaMonthly != "55.0000" {
		t.Errorf("total delta = %q, want 55.0000", d.TotalDeltaMonthly)
	}
	// canonicalized: largest |delta| first -> new (+35) then change (+30) then gone (-10)
	if d.Changes[0].Address != "new" {
		t.Errorf("first change = %q, want new", d.Changes[0].Address)
	}
}

func TestDiffNotEstimatedCarried(t *testing.T) {
	base := bd("0.0000", r("x", "0.0000", 0)) // not estimated (no components)
	prop := bd("0.0000", r("x", "0.0000", 0))
	d := diff.Differ{}.Diff(base, prop)
	if len(d.Changes) != 1 || !d.Changes[0].NotEstimated {
		t.Fatalf("expected NotEstimated carried: %+v", d.Changes)
	}
	if d.Changes[0].DeltaMonthly != "0" {
		t.Errorf("not-estimated delta should be 0, got %q", d.Changes[0].DeltaMonthly)
	}
}

// PBT-03: sum of estimated deltas equals total delta.
func TestDiffDeltaSumInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 8).Draw(t, "n")
		var baseR, propR []schema.Resource
		baseTotal, propTotal := new(big.Rat), new(big.Rat)
		for i := 0; i < n; i++ {
			addr := "r" + rapid.StringMatching(`[a-z]{1,3}`).Draw(t, "a")
			inBase := rapid.Bool().Draw(t, "ib")
			inProp := rapid.Bool().Draw(t, "ip")
			if !inBase && !inProp {
				inProp = true
			}
			bc := rapid.Int64Range(0, 100000).Draw(t, "bc")
			pc := rapid.Int64Range(0, 100000).Draw(t, "pc")
			bm := schema.RatToString(new(big.Rat).SetFrac64(bc, 100), 4)
			pm := schema.RatToString(new(big.Rat).SetFrac64(pc, 100), 4)
			if inBase {
				baseR = append(baseR, r(addr, bm, 1))
				baseTotal.Add(baseTotal, new(big.Rat).SetFrac64(bc, 100))
			}
			if inProp {
				propR = append(propR, r(addr, pm, 1))
				propTotal.Add(propTotal, new(big.Rat).SetFrac64(pc, 100))
			}
		}
		base := bd(schema.RatToString(baseTotal, 4), baseR...)
		prop := bd(schema.RatToString(propTotal, 4), propR...)

		d := diff.Differ{}.Diff(base, prop)

		sum := new(big.Rat)
		for _, c := range d.Changes {
			if c.NotEstimated {
				continue
			}
			r, _ := schema.StringToRat(c.DeltaMonthly)
			sum.Add(sum, r)
		}
		want, _ := schema.StringToRat(d.TotalDeltaMonthly)
		diff := new(big.Rat).Sub(sum, want)
		diff.Abs(diff)
		if diff.Cmp(big.NewRat(1, 100)) > 0 {
			t.Fatalf("Σ deltas %s != total delta %s", sum.FloatString(4), d.TotalDeltaMonthly)
		}
	})
}
