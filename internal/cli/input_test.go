package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassify(t *testing.T) {
	dir := t.TempDir()

	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(`{"format_version":"1.2","resource_changes":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	priorFile := filepath.Join(dir, "prior.json")
	if err := os.WriteFile(priorFile, []byte(`{"schema_version":"1.0","currency":"USD","resources":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	otherFile := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(otherFile, []byte(`hello`), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		in   string
		want InputKind
	}{
		{"", InputStdin},
		{"-", InputStdin},
		{dir, InputTerraformDir},
		{planFile, InputPlanFile},
		{priorFile, InputPriorJSON},
		{otherFile, InputPlanFile}, // fall through; parser will error
	}
	for _, tc := range tests {
		got, err := Classify(tc.in)
		if got != tc.want {
			t.Errorf("Classify(%q) = %v (err %v), want %v", tc.in, got, err, tc.want)
		}
	}

	if _, err := Classify(filepath.Join(dir, "nope.json")); err == nil {
		t.Error("missing file should return an error")
	}
}
