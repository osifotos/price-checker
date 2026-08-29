package usage

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// NFR-5.3: internal/usage depends only on stdlib, internal/plan, pkg/schema, and
// yaml (transitively via schema). It must not import internal/pricing or the SDK.
func TestImportsAreMinimal(t *testing.T) {
	forbidden := []string{"aws-sdk-go", "internal/pricing", "internal/engine", "internal/estimator"}
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
			for _, bad := range forbidden {
				if strings.Contains(path, bad) {
					t.Errorf("%s imports forbidden package %q", name, path)
				}
			}
		}
	}
}
