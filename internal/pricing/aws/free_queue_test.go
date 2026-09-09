package aws

import (
	"testing"

	"github.com/osifotos/price-checker/internal/pricing"
	"github.com/osifotos/price-checker/pkg/schema"
)

func TestSQSQueuePricerHappyPath(t *testing.T) {
	q := fakeQuerier{byPurpose: map[string]string{"sqs requests": "0.0000004"}}
	res := mkResource("aws_sqs_queue", map[string]any{}, nil)
	out := priceResource(t, sqsQueuePricer{}, res, schema.ResourceUsage{"monthly_requests": 1_000_000}, q)

	if !hasComponent(out, "Requests") {
		t.Fatalf("want Requests component, got %+v", out.Components)
	}
	if got := pricing.SumMonthly(out.Components); got != "0.4000" {
		t.Errorf("monthly cost = %q, want 0.4000", got)
	}
}

func TestSQSQueuePricerNoUsage(t *testing.T) {
	q := fakeQuerier{byPurpose: map[string]string{"sqs requests": "0.0000004"}}
	res := mkResource("aws_sqs_queue", map[string]any{}, nil)
	out := priceResource(t, sqsQueuePricer{}, res, schema.ResourceUsage{}, q)

	if reason, ok := notEstimatedReason(out, "Requests"); !ok || reason != schema.ReasonNoUsageData {
		t.Errorf("want NO_USAGE_DATA, got %+v", out.NotEstimated)
	}
}

func TestFreeResourcePricer(t *testing.T) {
	p := freeResourcePricer{}
	for _, typ := range p.ResourceTypes() {
		res := mkResource(typ, map[string]any{}, nil)
		out := priceResource(t, p, res, schema.ResourceUsage{}, fakeQuerier{})
		if len(out.Components) != 0 {
			t.Errorf("%s: want no cost components, got %+v", typ, out.Components)
		}
		if len(out.NotEstimated) != 1 || out.NotEstimated[0].ReasonCode != schema.ReasonNotBillable {
			t.Errorf("%s: want NOT_BILLABLE, got %+v", typ, out.NotEstimated)
		}
	}
}
