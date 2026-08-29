package aws

import (
	"context"
	"math/big"

	"github.com/example/price-checker/internal/pricing"
	"github.com/example/price-checker/pkg/schema"
)

// ---- aws_eks_cluster ----

type eksClusterPricer struct{}

func (eksClusterPricer) ResourceTypes() []string { return []string{"aws_eks_cluster"} }

func (eksClusterPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource
	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonEKS", RegionCode: in.Region, Purpose: "eks cluster",
		Filters: []pricing.Filter{f("usagetype", regionalUsageType(in.Region, "AmazonEKS-Hours:perCluster"))},
	}); {
	case err != nil:
		out.Skip(res, "EKS cluster", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "EKS cluster", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.HourlyComponent("EKS cluster", dim.USD); ok {
			out.Add(c)
		}
	}
	return out, nil
}

// ---- aws_eks_node_group ----

type eksNodeGroupPricer struct{}

func (eksNodeGroupPricer) ResourceTypes() []string { return []string{"aws_eks_node_group"} }

func (eksNodeGroupPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	instanceType := "t3.medium"
	if its, ok := res.Attributes.StringSlice("instance_types"); ok && len(its) > 0 {
		instanceType = its[0]
	}
	desired := int64(1)
	if sc, ok := res.Attributes.Block("scaling_config"); ok {
		if d, ok := sc.Float("desired_size"); ok && d > 0 {
			desired = int64(d)
		}
	}

	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonEC2", RegionCode: in.Region, Purpose: "eks node " + instanceType,
		Filters: []pricing.Filter{
			f("instanceType", instanceType),
			f("tenancy", "Shared"), f("operatingSystem", "Linux"),
			f("preInstalledSw", "NA"), f("capacitystatus", "Used"),
		},
	}); {
	case err != nil:
		out.Skip(res, "Node instances", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "Node instances", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.Component("Node instances ("+instanceType+" x"+i64(desired)+")", "instance-hours", dim.USD, big.NewRat(schema.MonthlyHours*desired, 1), false); ok {
			out.Add(c)
		}
	}
	return out, nil
}
