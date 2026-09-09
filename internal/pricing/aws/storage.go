package aws

import (
	"context"

	"github.com/osifotos/price-checker/internal/pricing"
	"github.com/osifotos/price-checker/pkg/schema"
)

// ---- aws_s3_bucket ----

type s3Pricer struct{}

func (s3Pricer) ResourceTypes() []string { return []string{"aws_s3_bucket"} }

func (s3Pricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	if gb, ok := in.Usage["storage_gb"]; ok {
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AmazonS3", RegionCode: in.Region, Purpose: "s3 storage",
			Filters: []pricing.Filter{f("productFamily", "Storage"), f("volumeType", "Standard"), f("storageClass", "General Purpose")},
		}); err == nil && found {
			if c, ok := pricing.Component("Storage (Standard)", "GB-months", dim.USD, pricing.RatFromFloat(gb), true); ok {
				out.Add(c)
			}
		}
	} else {
		out.Skip(res, "Storage", schema.ReasonNoUsageData)
	}

	if n, ok := in.Usage["monthly_tier1_requests"]; ok {
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AmazonS3", RegionCode: in.Region, Purpose: "s3 put requests",
			Filters: []pricing.Filter{f("productFamily", "API Request"), f("group", "S3-API-Tier1")},
		}); err == nil && found {
			if c, ok := pricing.Component("PUT/COPY/POST/LIST requests", "requests", dim.USD, pricing.RatFromFloat(n), true); ok {
				out.Add(c)
			}
		}
	} else {
		out.Skip(res, "Tier 1 requests", schema.ReasonNoUsageData)
	}

	if n, ok := in.Usage["monthly_tier2_requests"]; ok {
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AmazonS3", RegionCode: in.Region, Purpose: "s3 get requests",
			Filters: []pricing.Filter{f("productFamily", "API Request"), f("group", "S3-API-Tier2")},
		}); err == nil && found {
			if c, ok := pricing.Component("GET/SELECT requests", "requests", dim.USD, pricing.RatFromFloat(n), true); ok {
				out.Add(c)
			}
		}
	} else {
		out.Skip(res, "Tier 2 requests", schema.ReasonNoUsageData)
	}

	return out, nil
}
