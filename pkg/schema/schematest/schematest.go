// Package schematest provides fully populated sample documents for use in the
// tests of price-checker units that consume pkg/schema. It is test-support code
// but is a normal package so it can be imported across units.
package schematest

import "github.com/osifotos/price-checker/pkg/schema"

// SampleBreakdown returns a canonicalized Breakdown covering estimated and
// not-estimated resources, modules, and populated metadata.
func SampleBreakdown() *schema.Breakdown {
	b := schema.NewBreakdown(schema.PeriodMonth)
	b.Resources = []schema.Resource{
		{
			Address: "aws_instance.web", Type: "aws_instance", Name: "web",
			Module: "", Region: "us-east-1",
			MonthlyCost: "32.7680", HourlyCost: "0.0449",
			CostComponents: []schema.CostComponent{
				{Name: "Instance usage (t3.medium)", Unit: "hours", PricePerUnit: "0.0416000000",
					MonthlyQuantity: "730", MonthlyCost: "30.3680", HourlyCost: "0.0416", UsageBased: false},
				{Name: "root_block_device (gp3)", Unit: "GB-months", PricePerUnit: "0.0800000000",
					MonthlyQuantity: "30", MonthlyCost: "2.4000", HourlyCost: "0.0033", UsageBased: false},
			},
		},
		{
			Address: "module.net.aws_nat_gateway.this", Type: "aws_nat_gateway", Name: "this",
			Module: "module.net", Region: "us-east-1",
			MonthlyCost: "32.8500", HourlyCost: "0.0450",
			CostComponents: []schema.CostComponent{
				{Name: "NAT gateway", Unit: "hours", PricePerUnit: "0.0450000000",
					MonthlyQuantity: "730", MonthlyCost: "32.8500", HourlyCost: "0.0450", UsageBased: false},
			},
		},
	}
	b.Modules = []schema.ModuleTotal{
		{Module: "", MonthlyCost: "32.7680", HourlyCost: "0.0449", ResourceQty: 1},
		{Module: "module.net", MonthlyCost: "32.8500", HourlyCost: "0.0450", ResourceQty: 1},
	}
	b.NotEstimated = []schema.NotEstimated{
		{Address: "aws_s3_bucket.assets", Type: "aws_s3_bucket", Component: "Storage",
			ReasonCode: schema.ReasonNoUsageData, Message: schema.ReasonNoUsageData.Message()},
		{Address: "aws_kms_key.main", Type: "aws_kms_key",
			ReasonCode: schema.ReasonUnsupportedType, Message: schema.ReasonUnsupportedType.Message()},
	}
	b.TotalMonthly = "65.6180"
	b.TotalHourly = "0.0899"
	b.Summary = schema.CoverageSummary{
		ResourcesTotal: 4, ResourcesEstimated: 2, ResourcesNotEstimated: 2,
		ComponentsEstimated: 3, ComponentsNotEstimated: 1,
	}
	b.Metadata = schema.RunMetadata{
		ToolVersion: "0.0.0-test", GeneratedAt: "2026-08-29T00:00:00Z",
		PlanFormatVersion: "1.2", Regions: []string{"us-east-1"},
		UsageFileUsed: false, PriceQueries: 5, CacheHitRatio: 0.8, DedupHits: 2,
	}
	b.Canonicalize()
	return b
}

// SampleDiff returns a canonicalized DiffResult with added / removed / changed
// resources.
func SampleDiff() *schema.DiffResult {
	d := schema.NewDiffResult(schema.PeriodMonth)
	d.Changes = []schema.ResourceDelta{
		{Address: "aws_instance.web", Type: "aws_instance", Kind: schema.DeltaChanged,
			PriorMonthly: "30.6600", NewMonthly: "61.3200", DeltaMonthly: "30.6600", DeltaHourly: "0.0420"},
		{Address: "aws_nat_gateway.this", Type: "aws_nat_gateway", Kind: schema.DeltaAdded,
			PriorMonthly: "0", NewMonthly: "32.8500", DeltaMonthly: "32.8500", DeltaHourly: "0.0450"},
		{Address: "aws_eip.old", Type: "aws_eip", Kind: schema.DeltaRemoved,
			PriorMonthly: "3.6500", NewMonthly: "0", DeltaMonthly: "-3.6500", DeltaHourly: "-0.0050"},
	}
	d.TotalPriorMonthly = "34.3100"
	d.TotalNewMonthly = "94.1700"
	d.TotalDeltaMonthly = "59.8600"
	d.TotalDeltaHourly = "0.0820"
	d.Summary = schema.CoverageSummary{ResourcesTotal: 3, ResourcesEstimated: 3}
	d.Metadata = schema.RunMetadata{
		ToolVersion: "0.0.0-test", GeneratedAt: "2026-08-29T00:00:00Z",
		PlanFormatVersion: "1.2", Regions: []string{"us-east-1"},
	}
	d.Canonicalize()
	return d
}

// SampleUsage returns a usage file with two resources.
func SampleUsage() schema.UsageFile {
	return schema.UsageFile{
		Version: "0.1",
		ResourceUsage: map[string]schema.ResourceUsage{
			"aws_s3_bucket.assets":            {"storage_gb": 512, "monthly_tier1_requests": 100000},
			"module.net.aws_nat_gateway.this": {"monthly_data_processed_gb": 250},
		},
	}
}
