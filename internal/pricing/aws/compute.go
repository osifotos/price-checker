package aws

import (
	"context"
	"math/big"

	"github.com/osifotos/price-checker/internal/plan"
	"github.com/osifotos/price-checker/internal/pricing"
	"github.com/osifotos/price-checker/pkg/schema"
)

// ---- aws_instance ----

type instancePricer struct{}

func (instancePricer) ResourceTypes() []string { return []string{"aws_instance"} }

func (instancePricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	instanceType, ok := res.Attributes.String("instance_type")
	if !ok {
		out.Skip(res, "Instance usage", missingOrUnknown(res.Attributes, "instance_type"))
		return out, nil
	}

	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonEC2",
		RegionCode:  in.Region,
		Purpose:     "ec2 instance " + instanceType,
		Filters: []pricing.Filter{
			f("instanceType", instanceType),
			f("tenancy", "Shared"),
			f("operatingSystem", "Linux"),
			f("preInstalledSw", "NA"),
			f("capacitystatus", "Used"),
		},
	}); {
	case err != nil:
		out.Skip(res, "Instance usage", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "Instance usage", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.HourlyComponent("Instance usage (Linux/UNIX, "+instanceType+")", dim.USD); ok {
			out.Add(c)
		}
	}

	if rbd, ok := res.Attributes.Block("root_block_device"); ok {
		addBlockDeviceStorage(ctx, &out, in, rbd, "root_block_device")
	}
	for _, ebd := range res.Attributes.Blocks("ebs_block_device") {
		addBlockDeviceStorage(ctx, &out, in, ebd, "ebs_block_device")
	}
	return out, nil
}

func addBlockDeviceStorage(ctx context.Context, out *pricing.PricingOutput, in pricing.PricingInput, blk plan.AttrMap, label string) {
	res := in.Resource
	volType := "gp3"
	if v, ok := blk.String("volume_type"); ok {
		volType = v
	}
	sizeGB, ok := blk.Float("volume_size")
	if !ok {
		sizeGB = 8
	}
	dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonEC2", RegionCode: in.Region, Purpose: "ebs storage " + volType,
		Filters: []pricing.Filter{f("productFamily", "Storage"), f("volumeApiName", volType)},
	})
	switch {
	case err != nil:
		out.Skip(res, label+" storage", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, label+" storage", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.Component(label+" storage ("+volType+")", "GB-months", dim.USD, big.NewRat(int64(sizeGB), 1), false); ok {
			out.Add(c)
		}
	}
}

// ---- aws_ebs_volume ----

type ebsVolumePricer struct{}

func (ebsVolumePricer) ResourceTypes() []string { return []string{"aws_ebs_volume"} }

func (ebsVolumePricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	volType := "gp3"
	if v, ok := res.Attributes.String("type"); ok {
		volType = v
	}
	size, ok := res.Attributes.Float("size")
	if !ok {
		out.Skip(res, "Storage", missingOrUnknown(res.Attributes, "size"))
		return out, nil
	}

	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonEC2", RegionCode: in.Region, Purpose: "ebs " + volType,
		Filters: []pricing.Filter{f("productFamily", "Storage"), f("volumeApiName", volType)},
	}); {
	case err != nil:
		out.Skip(res, "Storage", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "Storage", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.Component("Storage ("+volType+")", "GB-months", dim.USD, big.NewRat(int64(size), 1), false); ok {
			out.Add(c)
		}
	}

	if iops, ok := res.Attributes.Float("iops"); ok && iops > 0 && (volType == "io1" || volType == "io2" || volType == "gp3") {
		if dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
			ServiceCode: "AmazonEC2", RegionCode: in.Region, Purpose: "ebs iops " + volType,
			Filters: []pricing.Filter{f("productFamily", "System Operation"), f("volumeApiName", volType), f("group", "EBS IOPS")},
		}); err == nil && found {
			billable := iops
			if volType == "gp3" {
				billable = iops - 3000
			}
			if billable > 0 {
				if c, ok := pricing.Component("Provisioned IOPS ("+volType+")", "IOPS-months", dim.USD, big.NewRat(int64(billable), 1), false); ok {
					out.Add(c)
				}
			}
		}
	}
	return out, nil
}

// ---- aws_ebs_snapshot / aws_ebs_snapshot_copy ----

type ebsSnapshotPricer struct{}

func (ebsSnapshotPricer) ResourceTypes() []string {
	return []string{"aws_ebs_snapshot", "aws_ebs_snapshot_copy"}
}

func (ebsSnapshotPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource

	sizeGB, ok := in.Usage["snapshot_size_gb"]
	if !ok {
		out.Skip(res, "Snapshot storage", schema.ReasonNoUsageData)
		return out, nil
	}
	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonEC2", RegionCode: in.Region, Purpose: "ebs snapshot",
		Filters: []pricing.Filter{f("productFamily", "Storage Snapshot")},
	}); {
	case err != nil:
		out.Skip(res, "Snapshot storage", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "Snapshot storage", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.Component("Snapshot storage", "GB-months", dim.USD, pricing.RatFromFloat(sizeGB), true); ok {
			out.Add(c)
		}
	}
	return out, nil
}

// ---- aws_eip ----

type eipPricer struct{}

func (eipPricer) ResourceTypes() []string { return []string{"aws_eip"} }

func (eipPricer) Price(ctx context.Context, in pricing.PricingInput) (pricing.PricingOutput, error) {
	var out pricing.PricingOutput
	res := in.Resource
	switch dim, found, err := query1(ctx, in.Q, pricing.PriceQuery{
		ServiceCode: "AmazonVPC", RegionCode: in.Region, Purpose: "public ipv4",
		Filters: []pricing.Filter{f("productFamily", "IP Address"), f("group", "VPCPublicIPv4Address")},
	}); {
	case err != nil:
		out.Skip(res, "Public IPv4 address", schema.ReasonPricingAPIError)
	case !found:
		out.Skip(res, "Public IPv4 address", schema.ReasonUnsupportedConfiguration)
	default:
		if c, ok := pricing.HourlyComponent("Public IPv4 address", dim.USD); ok {
			out.Add(c)
		}
	}
	return out, nil
}

// missingOrUnknown returns the right reason code for an absent attribute.
func missingOrUnknown(a plan.AttrMap, key string) schema.ReasonCode {
	if !a.IsKnown(key) {
		return schema.ReasonUnknownAfterApply
	}
	return schema.ReasonMissingAttribute
}
