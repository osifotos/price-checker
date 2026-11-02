package deprecation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/osifotos/price-checker/internal/plan"
)

func TestFindCombinesBuiltinAndProviderDeprecations(t *testing.T) {
	resources := []plan.Resource{
		{Address: "aws_launch_configuration.old", Type: "aws_launch_configuration"},
		{Address: "aws_old_resource.example", Type: "aws_old_resource"},
	}
	diagnostics := []plan.TerraformDeprecation{{
		Summary:  "Deprecated resource type",
		Detail:   "aws_old_resource is deprecated; use aws_new_resource",
		Filename: "main.tf", Line: 12,
	}}

	findings, err := Find(resources, diagnostics, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2: %+v", len(findings), findings)
	}
	if findings[0].Source != "aws-lifecycle" || findings[0].Replacement != "aws_launch_template" {
		t.Errorf("unexpected AWS lifecycle finding: %+v", findings[0])
	}
	if findings[1].Source != "terraform-provider" || findings[1].Address != "aws_old_resource.example" || findings[1].Line != 12 {
		t.Errorf("unexpected provider finding: %+v", findings[1])
	}
}

func TestFindLoadsCustomRulesAndOverridesBuiltins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.yml")
	data := []byte("resources:\n  aws_launch_configuration:\n    message: Custom retirement guidance.\n    replacement: aws_launch_template\n    effective_date: '2025-01-01'\n    reference: https://example.com/retirement\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	findings, err := Find([]plan.Resource{{Address: "aws_launch_configuration.old", Type: "aws_launch_configuration"}}, nil, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Message != "Custom retirement guidance." || findings[0].EffectiveDate != "2025-01-01" {
		t.Fatalf("custom rule did not override the builtin: %+v", findings)
	}
}
