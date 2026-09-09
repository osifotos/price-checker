package aws

import (
	"context"

	"github.com/osifotos/price-checker/internal/pricing"
	"github.com/osifotos/price-checker/pkg/schema"
)

// ---- aws_sqs_queue ----

type sqsQueuePricer struct{}

func (sqsQueuePricer) ResourceTypes() []string { return []string{"aws_sqs_queue"} }

func (sqsQueuePricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	queueType := "Standard"
	if fifo, ok := res.Attributes.Bool("fifo_queue"); ok && fifo {
		queueType = "FIFO (first-in, first-out)"
	}

	reqs, ok := in.Usage["monthly_requests"]
	if !ok {
		out.Skip(res, "Requests", schema.ReasonNoUsageData)
		return out, nil
	}

	if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AWSQueueService", RegionCode: in.Region, Purpose: "sqs requests",
		Filters: []pricing.Filter{f("productFamily", "API Request"), f("queueType", queueType)},
	}); err == nil && found {
		if c, ok := pricing.Component("Requests", "requests", dim.USD, pricing.RatFromFloat(reqs), true); ok {
			out.Add(c)
		}
	}

	return out, nil
}
