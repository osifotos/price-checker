package render_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/osifotos/price-checker/internal/render"
	"github.com/osifotos/price-checker/pkg/schema"
	"github.com/osifotos/price-checker/pkg/schema/schematest"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestRendererForKnownFormats(t *testing.T) {
	for _, f := range []string{"", "table", "json", "html", "github-comment"} {
		r, dr, err := render.RendererFor(f)
		if err != nil || r == nil || dr == nil {
			t.Errorf("RendererFor(%q) = %v %v %v", f, r, dr, err)
		}
	}
	if _, _, err := render.RendererFor("xml"); err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestTableRenderer(t *testing.T) {
	b := schematest.SampleBreakdown()
	var buf bytes.Buffer
	if err := (mustRenderer(t, "table")).Render(&buf, b, render.Options{Color: false, ShowComponents: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "PROJECT TOTAL") {
		t.Errorf("missing total line:\n%s", out)
	}
	if !strings.Contains(out, "NOT ESTIMATED (2)") {
		t.Errorf("missing not-estimated section:\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("Color=false output contains ANSI escapes:\n%q", out)
	}
	if !strings.Contains(out, "  - ") {
		t.Errorf("ShowComponents should indent components:\n%s", out)
	}
}

func TestJSONRendererValidatesAgainstSchema(t *testing.T) {
	b := schematest.SampleBreakdown()
	var buf bytes.Buffer
	if err := (mustRenderer(t, "json")).Render(&buf, b, render.Options{}); err != nil {
		t.Fatal(err)
	}
	compileAndValidate(t, schema.BreakdownSchemaJSON, buf.Bytes())

	// round trip
	var back schema.Breakdown
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatal(err)
	}
	back.Canonicalize()
	b2, _ := json.Marshal(&back)
	b1, _ := json.Marshal(b)
	if string(b1) != string(b2) {
		t.Errorf("json render not a faithful round trip")
	}
	if bytes.Contains(buf.Bytes(), []byte("\\u003c")) {
		t.Error("SetEscapeHTML(false) expected")
	}
}

func TestHTMLRendererSelfContained(t *testing.T) {
	b := schematest.SampleBreakdown()
	b.Resources = append(b.Resources, schema.Resource{
		Address: `aws_instance.<script>alert(1)</script>`, Type: "aws_instance", Name: "x",
		MonthlyCost: "1.0000", HourlyCost: "0.0014",
	})
	b.Canonicalize()

	var buf bytes.Buffer
	if err := (mustRenderer(t, "html")).Render(&buf, b, render.Options{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, bad := range []string{"http://", "https://", `src="//`, `href="//`} {
		if strings.Contains(out, bad) {
			t.Errorf("HTML report is not self-contained: found %q", bad)
		}
	}
	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Error("hostile resource name not escaped")
	}
	if !strings.Contains(out, "price-checker") {
		t.Error("missing report body")
	}
}

func TestGitHubCommentRenderer(t *testing.T) {
	b := schematest.SampleBreakdown()
	var buf1, buf2 bytes.Buffer
	r := mustRenderer(t, "github-comment")
	_ = r.Render(&buf1, b, render.Options{})
	_ = r.Render(&buf2, b, render.Options{})
	if buf1.String() != buf2.String() {
		t.Error("github-comment output is not deterministic")
	}
	out := buf1.String()
	if !strings.Contains(out, "estimated monthly cost") {
		t.Errorf("missing headline:\n%s", out)
	}
	if !strings.Contains(out, "<details>") {
		t.Errorf("missing details block:\n%s", out)
	}
	if !strings.Contains(out, "not estimated") {
		t.Errorf("missing not-estimated note:\n%s", out)
	}
}

func TestDiffRenderers(t *testing.T) {
	d := schematest.SampleDiff()
	for _, f := range []string{"table", "json", "html", "github-comment"} {
		_, dr, err := render.RendererFor(f)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := dr.RenderDiff(&buf, d, render.Options{}); err != nil {
			t.Errorf("%s RenderDiff: %v", f, err)
		}
		if buf.Len() == 0 {
			t.Errorf("%s produced empty diff output", f)
		}
	}
}

func mustRenderer(t *testing.T, f string) render.Renderer {
	t.Helper()
	r, _, err := render.RendererFor(f)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func compileAndValidate(t *testing.T, schemaDoc, payload []byte) {
	t.Helper()
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	if err := c.AddResource("s.json", bytes.NewReader(schemaDoc)); err != nil {
		t.Fatal(err)
	}
	sch, err := c.Compile("s.json")
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(payload, &v); err != nil {
		t.Fatal(err)
	}
	if err := sch.Validate(v); err != nil {
		t.Fatalf("payload violates schema: %v", err)
	}
}
