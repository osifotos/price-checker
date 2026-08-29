package aws

import (
	"context"
	"math/big"

	"github.com/example/price-checker/internal/pricing"
	"github.com/example/price-checker/pkg/schema"
)

// ---- aws_lambda_function ----

type lambdaPricer struct{}

func (lambdaPricer) ResourceTypes() []string { return []string{"aws_lambda_function"} }

func (lambdaPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	memoryMB := 128.0
	if m, ok := res.Attributes.Float("memory_size"); ok && m > 0 {
		memoryMB = m
	}

	reqs, hasReqs := in.Usage["monthly_requests"]
	durationMS, hasDur := in.Usage["request_duration_ms"]

	if !hasReqs || !hasDur {
		out.Skip(res, "Requests", schema.ReasonNoUsageData)
		out.Skip(res, "Duration (GB-seconds)", schema.ReasonNoUsageData)
	} else {
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AWSLambda", RegionCode: in.Region, Purpose: "lambda requests",
			Filters: []pricing.Filter{f("group", "AWS-Lambda-Requests")},
		}); err == nil && found {
			if c, ok := pricing.Component("Requests", "requests", dim.USD, pricing.RatFromFloat(reqs), true); ok {
				out.Add(c)
			}
		}
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AWSLambda", RegionCode: in.Region, Purpose: "lambda duration",
			Filters: []pricing.Filter{f("group", "AWS-Lambda-Duration")},
		}); err == nil && found {
			// GB-seconds = requests * (durationMS/1000) * (memoryMB/1024)
			gbSeconds := new(big.Rat).Mul(pricing.RatFromFloat(reqs), pricing.RatFromFloat(durationMS/1000.0))
			gbSeconds.Mul(gbSeconds, pricing.RatFromFloat(memoryMB/1024.0))
			if c, ok := pricing.Component("Duration (GB-seconds)", "GB-seconds", dim.USD, gbSeconds, true); ok {
				out.Add(c)
			}
		}
	}

	// provisioned concurrency
	return out, nil
}
