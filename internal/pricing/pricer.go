package pricing

import (
	"context"
	"math/big"

	"github.com/example/price-checker/internal/plan"
	"github.com/example/price-checker/pkg/schema"
)

// PricingInput is what a Pricer receives for one resource.
type PricingInput struct {
	Resource plan.Resource
	Usage    schema.ResourceUsage
	Region   string
	Q        PriceQuerier
}

// PricingOutput is what a Pricer returns for one resource.
type PricingOutput struct {
	Components   []schema.CostComponent
	NotEstimated []schema.NotEstimated
}

// Add appends a component.
func (o *PricingOutput) Add(c schema.CostComponent) { o.Components = append(o.Components, c) }

// Skip appends a not-estimated entry for the resource.
func (o *PricingOutput) Skip(res plan.Resource, component string, code schema.ReasonCode) {
	msg := code.Message()
	o.NotEstimated = append(o.NotEstimated, schema.NotEstimated{
		Address:    res.Address,
		Type:       res.Type,
		Component:  component,
		ReasonCode: code,
		Message:    msg,
	})
}

// Pricer encodes how one AWS resource type is priced. Implementations live in
// internal/pricing/aws and never import the AWS SDK.
type Pricer interface {
	ResourceTypes() []string
	Price(ctx context.Context, in PricingInput) (PricingOutput, error)
}

// ---- money helpers (the only place cost math happens) ----

const pricePrecision = 10
const costPrecision = 4

// Component builds a CostComponent from a per-unit USD price and a monthly
// quantity. hourly is derived as monthlyCost / 730.
func Component(name, unit, pricePerUnitUSD string, monthlyQty *big.Rat, usageBased bool) (schema.CostComponent, bool) {
	price, err := schema.StringToRat(pricePerUnitUSD)
	if err != nil {
		return schema.CostComponent{}, false
	}
	if monthlyQty == nil {
		monthlyQty = new(big.Rat)
	}
	monthlyCost := new(big.Rat).Mul(price, monthlyQty)
	hourlyCost := new(big.Rat).Quo(monthlyCost, big.NewRat(schema.MonthlyHours, 1))
	return schema.CostComponent{
		Name:            name,
		Unit:            unit,
		PricePerUnit:    schema.RatToString(price, pricePrecision),
		MonthlyQuantity: schema.RatToString(monthlyQty, costPrecision),
		MonthlyCost:     schema.RatToString(monthlyCost, costPrecision),
		HourlyCost:      schema.RatToString(hourlyCost, costPrecision),
		UsageBased:      usageBased,
	}, true
}

// HourlyComponent builds a component for something billed per hour for the whole
// month (730 hours).
func HourlyComponent(name, hourlyUSD string) (schema.CostComponent, bool) {
	return Component(name, "hours", hourlyUSD, big.NewRat(schema.MonthlyHours, 1), false)
}

// SumMonthly returns the decimal-string sum of the monthly costs of components.
func SumMonthly(cs []schema.CostComponent) string {
	total := new(big.Rat)
	for _, c := range cs {
		if r, err := schema.StringToRat(c.MonthlyCost); err == nil {
			total.Add(total, r)
		}
	}
	return schema.RatToString(total, costPrecision)
}

// SumHourly returns the decimal-string sum of the hourly costs of components.
func SumHourly(cs []schema.CostComponent) string {
	total := new(big.Rat)
	for _, c := range cs {
		if r, err := schema.StringToRat(c.HourlyCost); err == nil {
			total.Add(total, r)
		}
	}
	return schema.RatToString(total, costPrecision)
}

// RatFromDecimal parses a decimal string into *big.Rat (thin wrapper over
// schema.StringToRat) for use by pricers.
func RatFromDecimal(s string) (*big.Rat, error) { return schema.StringToRat(s) }

// RatFromFloat converts a usage quantity (float64) to *big.Rat exactly enough
// for pricing (6 decimal places).
func RatFromFloat(f float64) *big.Rat {
	r := new(big.Rat)
	if r.SetFloat64(f) == nil {
		return new(big.Rat)
	}
	return r
}
