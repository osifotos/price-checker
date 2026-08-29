package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/example/price-checker/pkg/schema"
	"github.com/example/price-checker/pkg/schema/schematest"
)

func TestBreakdownCanonicalizeDeterministic(t *testing.T) {
	a := schematest.SampleBreakdown()

	b := schematest.SampleBreakdown()
	reverse(b.Resources)
	reverseNE(b.NotEstimated)
	b.Canonicalize()

	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	if string(ab) != string(bb) {
		t.Fatalf("canonicalize not deterministic:\n a=%s\n b=%s", ab, bb)
	}
}

func TestDiffCanonicalizeOrdersByAbsDelta(t *testing.T) {
	d := schematest.SampleDiff() // already canonicalized
	want := []string{"aws_nat_gateway.this", "aws_instance.web", "aws_eip.old"}
	for i, w := range want {
		if d.Changes[i].Address != w {
			t.Fatalf("change %d = %s, want %s", i, d.Changes[i].Address, w)
		}
	}
}

func reverse(s []schema.Resource) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func reverseNE(s []schema.NotEstimated) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
