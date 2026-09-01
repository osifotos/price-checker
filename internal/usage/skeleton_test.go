package usage

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mtosin123/tf-price_checker/internal/plan"
)

func mkPlan(resources ...plan.Resource) *plan.Plan {
	raw := `{"format_version":"1.2","resource_changes":[`
	for i, r := range resources {
		if i > 0 {
			raw += ","
		}
		raw += `{"address":"` + r.Address + `","type":"` + r.Type + `","name":"` + r.Name +
			`","mode":"managed","module_address":"","change":{"actions":["create"],"after":{},"after_unknown":{}}}`
	}
	raw += `]}`
	p, err := plan.NewParser().Parse([]byte(raw))
	if err != nil {
		panic(err)
	}
	return p
}

func TestGenerateSkeleton(t *testing.T) {
	p := mkPlan(
		plan.Resource{Address: "aws_s3_bucket.assets", Type: "aws_s3_bucket", Name: "assets"},
		plan.Resource{Address: "aws_nat_gateway.n", Type: "aws_nat_gateway", Name: "n"},
		plan.Resource{Address: "aws_instance.web", Type: "aws_instance", Name: "web"}, // no usage keys
	)
	var buf bytes.Buffer
	if err := GenerateSkeleton(p, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, "aws_s3_bucket.assets:") {
		t.Errorf("missing s3 bucket block:\n%s", out)
	}
	if !strings.Contains(out, "storage_gb: 0") {
		t.Errorf("missing storage_gb key:\n%s", out)
	}
	if !strings.Contains(out, "monthly_data_processed_gb: 0") {
		t.Errorf("missing nat gateway key:\n%s", out)
	}
	if strings.Contains(out, "aws_instance.web") {
		t.Errorf("aws_instance has no usage keys and should be omitted:\n%s", out)
	}

	// round-trips through Load
	m, err := loadBytes(out)
	if err != nil {
		t.Fatalf("generated skeleton does not load: %v", err)
	}
	if len(m.Warnings()) != 0 {
		t.Errorf("skeleton produced warnings: %v", m.Warnings())
	}
}

func TestGenerateSkeletonEmpty(t *testing.T) {
	p := mkPlan(plan.Resource{Address: "aws_instance.web", Type: "aws_instance", Name: "web"})
	var buf bytes.Buffer
	if err := GenerateSkeleton(p, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "resource_usage: {}") {
		t.Errorf("expected empty resource_usage, got:\n%s", buf.String())
	}
	if _, err := loadBytes(buf.String()); err != nil {
		t.Fatalf("empty skeleton does not load: %v", err)
	}
}

// loadBytes writes content to a temp file and Loads it.
func loadBytes(content string) (Model, error) {
	f, err := writeTemp(content)
	if err != nil {
		return nil, err
	}
	return Load(f)
}
