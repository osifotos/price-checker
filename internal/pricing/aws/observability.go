package aws

import (
	"context"
	"math/big"

	"github.com/osifotos/price-checker/internal/pricing"
	"github.com/osifotos/price-checker/pkg/schema"
)

// ---- aws_cloudwatch_* ----

type cloudWatchPricer struct{}

func (cloudWatchPricer) ResourceTypes() []string {
	return []string{
		"aws_cloudwatch_log_group",
		"aws_cloudwatch_metric_alarm",
		"aws_cloudwatch_dashboard",
	}
}

func (cloudWatchPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	switch res.Type {
	case "aws_cloudwatch_metric_alarm":
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AmazonCloudWatch", RegionCode: in.Region, Purpose: "cw alarm",
			Filters: []pricing.Filter{f("productFamily", "Alarm"), f("alarmType", "Standard")},
		}); err == nil && found {
			if c, ok := pricing.Component("Standard alarm", "alarms", dim.USD, big.NewRat(1, 1), false); ok {
				out.Add(c)
			}
		} else {
			out.Skip(res, "Standard alarm", schema.ReasonUnsupportedConfiguration)
		}
	case "aws_cloudwatch_dashboard":
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AmazonCloudWatch", RegionCode: in.Region, Purpose: "cw dashboard",
			Filters: []pricing.Filter{f("productFamily", "Dashboard")},
		}); err == nil && found {
			if c, ok := pricing.Component("Dashboard", "dashboards", dim.USD, big.NewRat(1, 1), false); ok {
				out.Add(c)
			}
		} else {
			out.Skip(res, "Dashboard", schema.ReasonUnsupportedConfiguration)
		}
	case "aws_cloudwatch_log_group":
		out.Skip(res, "Log ingestion", schema.ReasonNoUsageData)
		out.Skip(res, "Log storage", schema.ReasonNoUsageData)
	}
	return out, nil
}
