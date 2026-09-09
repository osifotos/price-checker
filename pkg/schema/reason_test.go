package schema_test

import (
	"testing"

	"github.com/osifotos/price-checker/pkg/schema"
)

func TestReasonCodeValid(t *testing.T) {
	for _, c := range schema.ReasonCodes() {
		if !c.Valid() {
			t.Errorf("ReasonCodes() returned invalid code %q", c)
		}
		if c.Message() == "" {
			t.Errorf("reason code %q has no message", c)
		}
	}
	if schema.ReasonCode("NOPE").Valid() {
		t.Error("unknown reason code reported valid")
	}
	if len(schema.ReasonCodes()) != 7 {
		t.Fatalf("expected 7 reason codes, got %d", len(schema.ReasonCodes()))
	}
}

func TestReasonCodesSorted(t *testing.T) {
	got := schema.ReasonCodes()
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Fatalf("ReasonCodes() not sorted: %v", got)
		}
	}
}
