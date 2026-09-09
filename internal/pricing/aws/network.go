package aws

import (
	"context"

	"github.com/osifotos/price-checker/internal/pricing"
	"github.com/osifotos/price-checker/pkg/schema"
)

// ---- aws_lb / aws_alb / aws_elb ----

type loadBalancerPricer struct{}

func (loadBalancerPricer) ResourceTypes() []string {
	return []string{"aws_lb", "aws_alb", "aws_elb"}
}

func (loadBalancerPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	lbType := "application"
	if res.Type == "aws_elb" {
		lbType = "classic"
	} else if t, ok := res.Attributes.String("load_balancer_type"); ok {
		lbType = t
	}

	var usagetype, label, lcuKey string
	switch lbType {
	case "application":
		usagetype, label, lcuKey = "LoadBalancerUsage", "Application load balancer", "monthly_lcu"
	case "network":
		usagetype, label, lcuKey = "LoadBalancerUsage", "Network load balancer", "monthly_nlcu"
	case "gateway":
		usagetype, label, lcuKey = "LoadBalancerUsage", "Gateway load balancer", "monthly_glcu"
	default:
		usagetype, label, lcuKey = "LoadBalancerUsage", "Classic load balancer", ""
	}

	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: elbServiceCode(lbType), RegionCode: in.Region, Purpose: "elb " + lbType,
		Filters: []pricing.Filter{f("productFamily", "Load Balancer"+elbFamilySuffix(lbType)), f("usagetype", regionalUsageType(in.Region, usagetype))},
	}); {
	case err != nil:
		out.Skip(res, label, schema.ReasonPricingAPIError)
	case !found:
		// try without the usagetype filter as a fallback
		if dim2, found2, err2 := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: elbServiceCode(lbType), RegionCode: in.Region, Purpose: "elb fallback " + lbType,
			Filters: []pricing.Filter{f("productFamily", "Load Balancer"+elbFamilySuffix(lbType))},
		}); err2 == nil && found2 {
			if c, ok := pricing.HourlyComponent(label, dim2.USD); ok {
				out.Add(c)
			}
		} else {
			out.Skip(res, label, schema.ReasonUnsupportedConfiguration)
		}
	default:
		if c, ok := pricing.HourlyComponent(label, dim.USD); ok {
			out.Add(c)
		}
	}

	if lcuKey != "" {
		if _, ok := in.Usage[lcuKey]; !ok {
			out.Skip(res, "Load balancer capacity units", schema.ReasonNoUsageData)
		}
	}
	return out, nil
}

func elbServiceCode(lbType string) string {
	if lbType == "classic" {
		return "AWSELB"
	}
	return "AWSELB"
}
func elbFamilySuffix(lbType string) string {
	switch lbType {
	case "application":
		return "-Application"
	case "network":
		return "-Network"
	case "gateway":
		return "-Gateway"
	default:
		return ""
	}
}

// ---- aws_nat_gateway ----

type natGatewayPricer struct{}

func (natGatewayPricer) ResourceTypes() []string { return []string{"aws_nat_gateway"} }

func (natGatewayPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonVPC", RegionCode: in.Region, Purpose: "nat gateway hours",
		Filters: []pricing.Filter{f("productFamily", "NAT Gateway"), f("usagetype", regionalUsageType(in.Region, "NatGateway-Hours"))},
	}); {
	case err != nil:
		out.Skip(res, "NAT gateway", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "NAT gateway", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.HourlyComponent("NAT gateway", dim.USD); ok {
			out.Add(c)
		}
	}

	gb, ok := in.Usage["monthly_data_processed_gb"]
	if !ok {
		out.Skip(res, "Data processed", schema.ReasonNoUsageData)
		return out, nil
	}
	if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonVPC", RegionCode: in.Region, Purpose: "nat gateway data",
		Filters: []pricing.Filter{f("productFamily", "NAT Gateway"), f("usagetype", regionalUsageType(in.Region, "NatGateway-Bytes"))},
	}); err == nil && found {
		if c, ok := pricing.Component("Data processed", "GB", dim.USD, pricing.RatFromFloat(gb), true); ok {
			out.Add(c)
		}
	}
	return out, nil
}

// regionalUsageType prefixes a usage type with the region's billing code prefix
// (e.g. "USE1-") when the region is not us-east-1. This is a best effort; if the
// filter misses, callers fall back to a coarser query.
func regionalUsageType(region, base string) string {
	prefix := map[string]string{
		"us-east-2": "USE2-", "us-west-1": "USW1-", "us-west-2": "USW2-",
		"eu-west-1": "EUW1-", "eu-west-2": "EUW2-", "eu-central-1": "EUC1-",
		"ap-south-1": "APS3-", "ap-southeast-1": "APS1-", "ap-southeast-2": "APS2-",
		"ap-northeast-1": "APN1-",
	}[region]
	return prefix + base
}
