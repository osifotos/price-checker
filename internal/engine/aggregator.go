// Package engine folds priced resources into a schema.Breakdown, owning the
// decimal roll-ups and the coverage accounting.
package engine

import (
	"math/big"
	"sort"

	"github.com/mtosin123/tf-price_checker/internal/plan"
	"github.com/mtosin123/tf-price_checker/pkg/schema"
)

// PricedResource is one resource after its pricer has run.
type PricedResource struct {
	Resource     plan.Resource
	Region       string
	Components   []schema.CostComponent
	NotEstimated []schema.NotEstimated
}

// Aggregator builds a schema.Breakdown from priced resources.
type Aggregator struct{}

const costDP = 4

// Build produces the final, canonicalized breakdown.
func (Aggregator) Build(priced []PricedResource, meta schema.RunMetadata, period schema.Period) *schema.Breakdown {
	b := schema.NewBreakdown(period)
	b.Metadata = meta

	moduleMonthly := map[string]*big.Rat{}
	moduleHourly := map[string]*big.Rat{}
	moduleCount := map[string]int{}
	projMonthly := new(big.Rat)
	projHourly := new(big.Rat)

	var summary schema.CoverageSummary
	summary.ResourcesTotal = len(priced)

	regionSet := map[string]bool{}

	for _, pr := range priced {
		res := pr.Resource
		if pr.Region != "" {
			regionSet[pr.Region] = true
		}

		mRat := sumField(pr.Components, func(c schema.CostComponent) string { return c.MonthlyCost })
		hRat := sumField(pr.Components, func(c schema.CostComponent) string { return c.HourlyCost })
		monthly := schema.RatToString(mRat, costDP)
		hourly := schema.RatToString(hRat, costDP)

		b.Resources = append(b.Resources, schema.Resource{
			Address:        res.Address,
			Type:           res.Type,
			Name:           res.Name,
			Module:         res.ModuleAddress,
			Region:         pr.Region,
			MonthlyCost:    monthly,
			HourlyCost:     hourly,
			CostComponents: append([]schema.CostComponent(nil), pr.Components...),
		})

		addTo(moduleMonthly, res.ModuleAddress, mRat)
		addTo(moduleHourly, res.ModuleAddress, hRat)
		moduleCount[res.ModuleAddress]++
		projMonthly.Add(projMonthly, mRat)
		projHourly.Add(projHourly, hRat)

		b.NotEstimated = append(b.NotEstimated, pr.NotEstimated...)

		if len(pr.Components) > 0 {
			summary.ResourcesEstimated++
		} else {
			summary.ResourcesNotEstimated++
		}
		summary.ComponentsEstimated += len(pr.Components)
		summary.ComponentsNotEstimated += len(pr.NotEstimated)
	}

	for mod := range moduleCount {
		b.Modules = append(b.Modules, schema.ModuleTotal{
			Module:      mod,
			MonthlyCost: schema.RatToString(moduleMonthly[mod], costDP),
			HourlyCost:  schema.RatToString(moduleHourly[mod], costDP),
			ResourceQty: moduleCount[mod],
		})
	}
	sort.Slice(b.Modules, func(i, j int) bool { return b.Modules[i].Module < b.Modules[j].Module })

	b.TotalMonthly = schema.RatToString(projMonthly, costDP)
	b.TotalHourly = schema.RatToString(projHourly, costDP)
	b.Summary = summary

	if len(meta.Regions) == 0 {
		regions := make([]string, 0, len(regionSet))
		for r := range regionSet {
			regions = append(regions, r)
		}
		sort.Strings(regions)
		b.Metadata.Regions = regions
	}

	b.Canonicalize()
	return b
}

func sumField(cs []schema.CostComponent, pick func(schema.CostComponent) string) *big.Rat {
	total := new(big.Rat)
	for _, c := range cs {
		if r, err := schema.StringToRat(pick(c)); err == nil {
			total.Add(total, r)
		}
	}
	return total
}

func addTo(m map[string]*big.Rat, key string, v *big.Rat) {
	if acc, ok := m[key]; ok {
		acc.Add(acc, v)
		return
	}
	m[key] = new(big.Rat).Set(v)
}
