package schema_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mtosin123/tf-price_checker/pkg/schema"
	"github.com/mtosin123/tf-price_checker/pkg/schema/schematest"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

func compile(t *testing.T, id string, doc []byte) *jsonschema.Schema {
	t.Helper()
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	if err := c.AddResource(id, bytes.NewReader(doc)); err != nil {
		t.Fatalf("add schema %s: %v", id, err)
	}
	sch, err := c.Compile(id)
	if err != nil {
		t.Fatalf("compile schema %s: %v", id, err)
	}
	return sch
}

func validate(t *testing.T, sch *jsonschema.Schema, payload []byte) {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal(payload, &v); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if err := sch.Validate(v); err != nil {
		t.Fatalf("payload does not satisfy schema: %v\npayload: %s", err, payload)
	}
}

func TestBreakdownMatchesSchema(t *testing.T) {
	sch := compile(t, "breakdown.schema.json", schema.BreakdownSchemaJSON)
	out, err := json.Marshal(schematest.SampleBreakdown())
	if err != nil {
		t.Fatal(err)
	}
	validate(t, sch, out)
}

func TestEmptyBreakdownMatchesSchema(t *testing.T) {
	sch := compile(t, "breakdown.schema.json", schema.BreakdownSchemaJSON)
	b := schema.NewBreakdown(schema.PeriodHour)
	b.Metadata = schema.RunMetadata{
		ToolVersion: "x", GeneratedAt: "2026-08-29T00:00:00Z", PlanFormatVersion: "1.2",
		Regions: []string{},
	}
	b.Canonicalize()
	out, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	validate(t, sch, out)

	for _, frag := range []string{`"resources":[]`, `"modules":[]`, `"not_estimated":[]`} {
		if !bytes.Contains(out, []byte(frag)) {
			t.Errorf("expected %s in %s", frag, out)
		}
	}
}

func TestDiffMatchesSchema(t *testing.T) {
	sch := compile(t, "diff.schema.json", schema.DiffSchemaJSON)
	out, err := json.Marshal(schematest.SampleDiff())
	if err != nil {
		t.Fatal(err)
	}
	validate(t, sch, out)
}

func TestUsageMatchesSchema(t *testing.T) {
	sch := compile(t, "usage.schema.json", schema.UsageSchemaJSON)
	out, err := json.Marshal(schematest.SampleUsage())
	if err != nil {
		t.Fatal(err)
	}
	validate(t, sch, out)
}
