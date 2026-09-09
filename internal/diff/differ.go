// Package diff computes the cost change between two schema.Breakdown documents.
// Differ.Diff is a pure function and depends only on pkg/schema.
package diff

import (
	"math/big"
	"sort"

	"github.com/osifotos/price-checker/pkg/schema"
)

// Differ computes cost diffs.
type Differ struct{}

// Diff returns the change from base to proposed.
func (Differ) Diff(base, proposed *schema.Breakdown) *schema.DiffResult {
	d := schema.NewDiffResult(proposed.Period)

	baseByAddr := indexByAddress(base)
	propByAddr := indexByAddress(proposed)

	addrs := unionSorted(baseByAddr, propByAddr)

	totalDelta := new(big.Rat)
	totalDeltaHourly := new(big.Rat)
	var summary schema.CoverageSummary
	summary.ResourcesTotal = len(addrs)

	for _, addr := range addrs {
		bRes, inBase := baseByAddr[addr]
		pRes, inProp := propByAddr[addr]

		delta := schema.ResourceDelta{Address: addr}

		switch {
		case inBase && inProp:
			delta.Type = pRes.Type
			delta.PriorMonthly = norm(bRes.MonthlyCost)
			delta.NewMonthly = norm(pRes.MonthlyCost)
			if notEstimated(bRes) || notEstimated(pRes) {
				delta.NotEstimated = true
				delta.Kind = schema.DeltaUnchanged
				if delta.PriorMonthly != delta.NewMonthly {
					delta.Kind = schema.DeltaChanged
				}
			} else if delta.PriorMonthly == delta.NewMonthly {
				delta.Kind = schema.DeltaUnchanged
			} else {
				delta.Kind = schema.DeltaChanged
			}
		case inProp:
			delta.Type = pRes.Type
			delta.Kind = schema.DeltaAdded
			delta.PriorMonthly = "0"
			delta.NewMonthly = norm(pRes.MonthlyCost)
			delta.NotEstimated = notEstimated(pRes)
		default: // inBase only
			delta.Type = bRes.Type
			delta.Kind = schema.DeltaRemoved
			delta.PriorMonthly = norm(bRes.MonthlyCost)
			delta.NewMonthly = "0"
			delta.NotEstimated = notEstimated(bRes)
		}

		dm := sub(delta.NewMonthly, delta.PriorMonthly)
		dh := sub(hourlyOf(pRes, inProp), hourlyOf(bRes, inBase))
		if delta.NotEstimated {
			delta.DeltaMonthly = "0"
			delta.DeltaHourly = "0"
		} else {
			delta.DeltaMonthly = schema.RatToString(dm, 4)
			delta.DeltaHourly = schema.RatToString(dh, 4)
			totalDelta.Add(totalDelta, dm)
			totalDeltaHourly.Add(totalDeltaHourly, dh)
			summary.ResourcesEstimated++
		}
		d.Changes = append(d.Changes, delta)
	}

	summary.ResourcesNotEstimated = summary.ResourcesTotal - summary.ResourcesEstimated

	d.TotalPriorMonthly = norm(base.TotalMonthly)
	d.TotalNewMonthly = norm(proposed.TotalMonthly)
	d.TotalDeltaMonthly = schema.RatToString(totalDelta, 4)
	d.TotalDeltaHourly = schema.RatToString(totalDeltaHourly, 4)
	d.Summary = summary
	d.Metadata = proposed.Metadata

	d.Canonicalize()
	return d
}

func indexByAddress(b *schema.Breakdown) map[string]schema.Resource {
	m := make(map[string]schema.Resource, len(b.Resources))
	for _, r := range b.Resources {
		m[r.Address] = r
	}
	return m
}

func unionSorted(a, b map[string]schema.Resource) []string {
	seen := map[string]bool{}
	var out []string
	for k := range a {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	for k := range b {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// notEstimated reports whether a resource received no cost estimate.
func notEstimated(r schema.Resource) bool { return len(r.CostComponents) == 0 }

func hourlyOf(r schema.Resource, present bool) string {
	if !present {
		return "0"
	}
	return norm(r.HourlyCost)
}

func norm(s string) string {
	r, err := schema.StringToRat(s)
	if err != nil {
		return "0"
	}
	return schema.RatToString(r, 4)
}

func sub(a, b string) *big.Rat {
	ra, _ := schema.StringToRat(a)
	rb, _ := schema.StringToRat(b)
	if ra == nil {
		ra = new(big.Rat)
	}
	if rb == nil {
		rb = new(big.Rat)
	}
	return new(big.Rat).Sub(ra, rb)
}
