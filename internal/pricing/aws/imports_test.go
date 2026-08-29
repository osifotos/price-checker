package aws

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// NFR-5.3 / NFR-5.1: pricer implementations must never import the AWS SDK; all
// AWS access is through pricing.PriceQuerier.
func TestNoAWSSDKImport(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "aws-sdk-go") || strings.Contains(path, "aws/aws-sdk") {
				t.Errorf("%s imports the AWS SDK (%q) — pricers must use pricing.PriceQuerier", name, path)
			}
		}
	}
}
