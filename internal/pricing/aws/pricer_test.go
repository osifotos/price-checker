package aws

import (
	"context"
	"strings"
	"testing"

	"github.com/example/price-checker/internal/plan"
	"github.com/example/price-checker/internal/pricing"
	"github.com/example/price-checker/pkg/schema"
)

// fakeQuerier returns a price dimension when the query Purpose contains one of
// the configured substrings.
type fakeQuerier struct {
	byPurpose map[string]string // substring -> USD price
	err       error
}

func (f fakeQuerier) Query(ctx context.Context, q pricing.PriceQuery) ([]pricing.PriceDimension, error) {
	if f.err != nil {
		return nil, f.err
	}
	for sub, usd := range f.byPurpose {
		if strings.Contains(q.Purpose, sub) {
			return []pricing.PriceDimension{{SKU: "X", RateCode: "X.1", Unit: "Hrs", USD: usd}}, nil
		}
	}
	return nil, nil
}

func priceResource(t *testing.T, p pricing.Pricer, res plan.Resource, usage schema.ResourceUsage, q pricing.PriceQuerier) pricing.PricingOutput {
	t.Helper()
	out, err := p.Price(context.Background(), pricing.PricingInput{
		Resource: res, Usage: usage, Region: "us-east-1", Q: q,
	})
	if err != nil {
		t.Fatalf("Price returned error: %v", err)
	}
	return out
}

func mkResource(typ string, attrs map[string]any, unknown map[string]any) plan.Resource {
	return plan.Resource{
		Address:    typ + ".x",
		Type:       typ,
		Name:       "x",
		Mode:       "managed",
		Action:     plan.ActionCreate,
		Attributes: plan.NewAttrMapForTest(attrs, unknown),
	}
}

func hasComponent(out pricing.PricingOutput, nameSub string) bool {
	for _, c := range out.Components {
		if strings.Contains(c.Name, nameSub) {
			return true
		}
	}
	return false
}

func notEstimatedReason(out pricing.PricingOutput, compSub string) (schema.ReasonCode, bool) {
	for _, ne := range out.NotEstimated {
		if strings.Contains(ne.Component, compSub) {
			return ne.ReasonCode, true
		}
	}
	return "", false
}

func TestInstancePricerHappyPath(t *testing.T) {
	q := fakeQuerier{byPurpose: map[string]string{
		"ec2 instance t3.medium": "0.0416",
		"ebs storage gp3":        "0.08",
	}}
	res := mkResource("aws_instance", map[string]any{
		"instance_type":     "t3.medium",
		"root_block_device": []any{map[string]any{"volume_type": "gp3", "volume_size": float64(30)}},
	}, nil)
	out := priceResource(t, instancePricer{}, res, nil, q)
	if !hasComponent(out, "Instance usage") {
		t.Errorf("missing instance component: %+v", out.Components)
	}
	if !hasComponent(out, "root_block_device storage") {
		t.Errorf("missing storage component: %+v", out.Components)
	}
}

func TestInstancePricerMissingType(t *testing.T) {
	res := mkResource("aws_instance", map[string]any{}, map[string]any{"instance_type": true})
	out := priceResource(t, instancePricer{}, res, nil, fakeQuerier{})
	if code, ok := notEstimatedReason(out, "Instance usage"); !ok || code != schema.ReasonUnknownAfterApply {
		t.Errorf("want UNKNOWN_AFTER_APPLY, got %q %v", code, ok)
	}
}

func TestEBSVolumePricer(t *testing.T) {
	q := fakeQuerier{byPurpose: map[string]string{"ebs gp3": "0.08"}}
	res := mkResource("aws_ebs_volume", map[string]any{"type": "gp3", "size": float64(100)}, nil)
	out := priceResource(t, ebsVolumePricer{}, res, nil, q)
	if !hasComponent(out, "Storage (gp3)") {
		t.Errorf("missing storage: %+v", out.Components)
	}
}

func TestSnapshotPricerNoUsage(t *testing.T) {
	res := mkResource("aws_ebs_snapshot", map[string]any{}, nil)
	out := priceResource(t, ebsSnapshotPricer{}, res, nil, fakeQuerier{})
	if code, ok := notEstimatedReason(out, "Snapshot storage"); !ok || code != schema.ReasonNoUsageData {
		t.Errorf("want NO_USAGE_DATA, got %q %v", code, ok)
	}
}

func TestNatGatewayPricer(t *testing.T) {
	q := fakeQuerier{byPurpose: map[string]string{
		"nat gateway hours": "0.045",
		"nat gateway data":  "0.045",
	}}
	res := mkResource("aws_nat_gateway", map[string]any{}, nil)

	out := priceResource(t, natGatewayPricer{}, res, nil, q)
	if !hasComponent(out, "NAT gateway") {
		t.Errorf("missing NAT gateway hours: %+v", out.Components)
	}
	if _, ok := notEstimatedReason(out, "Data processed"); !ok {
		t.Errorf("expected Data processed to be not-estimated without usage")
	}

	out2 := priceResource(t, natGatewayPricer{}, res, schema.ResourceUsage{"monthly_data_processed_gb": 100}, q)
	if !hasComponent(out2, "Data processed") {
		t.Errorf("expected Data processed component with usage: %+v", out2.Components)
	}
}

func TestRDSInstancePricer(t *testing.T) {
	q := fakeQuerier{byPurpose: map[string]string{
		"rds db.t3.medium": "0.068",
		"rds storage":      "0.115",
	}}
	res := mkResource("aws_db_instance", map[string]any{
		"instance_class": "db.t3.medium", "engine": "postgres",
		"allocated_storage": float64(50), "multi_az": true,
	}, nil)
	out := priceResource(t, rdsInstancePricer{}, res, nil, q)
	if !hasComponent(out, "Database instance (Multi-AZ") {
		t.Errorf("missing multi-az db instance: %+v", out.Components)
	}
	if !hasComponent(out, "Database storage") {
		t.Errorf("missing db storage: %+v", out.Components)
	}
}

func TestLambdaPricerNeedsUsage(t *testing.T) {
	res := mkResource("aws_lambda_function", map[string]any{"memory_size": float64(512)}, nil)
	out := priceResource(t, lambdaPricer{}, res, nil, fakeQuerier{})
	if _, ok := notEstimatedReason(out, "Requests"); !ok {
		t.Errorf("expected Requests not-estimated without usage")
	}

	q := fakeQuerier{byPurpose: map[string]string{
		"lambda requests": "0.0000002",
		"lambda duration": "0.0000166667",
	}}
	out2 := priceResource(t, lambdaPricer{}, res,
		schema.ResourceUsage{"monthly_requests": 1_000_000, "request_duration_ms": 200}, q)
	if !hasComponent(out2, "Requests") || !hasComponent(out2, "Duration") {
		t.Errorf("expected requests+duration components: %+v", out2.Components)
	}
}

func TestPricerAPIErrorReason(t *testing.T) {
	q := fakeQuerier{err: context.DeadlineExceeded}
	res := mkResource("aws_instance", map[string]any{"instance_type": "t3.medium"}, nil)
	out := priceResource(t, instancePricer{}, res, nil, q)
	if code, ok := notEstimatedReason(out, "Instance usage"); !ok || code != schema.ReasonPricingAPIError {
		t.Errorf("want PRICING_API_ERROR, got %q %v", code, ok)
	}
}
