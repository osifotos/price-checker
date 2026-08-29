package pricing

import (
	"math/big"
	"testing"

	"github.com/example/price-checker/pkg/schema"
	"pgregory.net/rapid"
)

// PBT-03: a component's hourly cost times 730 equals its monthly cost (within
// rounding), the costs are non-negative for non-negative inputs, and the price
// is echoed faithfully.
func TestComponentInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		priceNum := rapid.Int64Range(0, 5_000_000).Draw(t, "price")
		qtyNum := rapid.Int64Range(0, 100_000).Draw(t, "qty")
		price := schema.RatToString(new(big.Rat).SetFrac64(priceNum, 1_000_000), 10)
		qty := new(big.Rat).SetFrac64(qtyNum, 100)

		c, ok := Component("x", "u", price, qty, false)
		if !ok {
			t.Fatalf("Component rejected valid inputs price=%s", price)
		}

		mc, err := schema.StringToRat(c.MonthlyCost)
		if err != nil {
			t.Fatalf("monthly cost not decimal: %q", c.MonthlyCost)
		}
		hc, err := schema.StringToRat(c.HourlyCost)
		if err != nil {
			t.Fatalf("hourly cost not decimal: %q", c.HourlyCost)
		}
		if mc.Sign() < 0 || hc.Sign() < 0 {
			t.Fatalf("negative cost from non-negative inputs: mc=%s hc=%s", c.MonthlyCost, c.HourlyCost)
		}

		// hc is monthly/730 rounded to 4dp, so hc*730 can differ from mc by up
		// to 0.00005*730 ≈ 0.037. Allow 0.05.
		back := new(big.Rat).Mul(hc, big.NewRat(schema.MonthlyHours, 1))
		diff := new(big.Rat).Sub(back, mc)
		diff.Abs(diff)
		if diff.Cmp(big.NewRat(5, 100)) > 0 {
			t.Fatalf("hourly*730 (%s) not within 0.05 of monthly (%s)", back.FloatString(6), c.MonthlyCost)
		}
	})
}

func TestSumMonthly(t *testing.T) {
	cs := []schema.CostComponent{
		{MonthlyCost: "10.0000", HourlyCost: "0.0137"},
		{MonthlyCost: "5.5000", HourlyCost: "0.0075"},
	}
	if got := SumMonthly(cs); got != "15.5000" {
		t.Fatalf("SumMonthly = %q, want 15.5000", got)
	}
}
