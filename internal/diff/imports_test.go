package diff

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// NFR-5.3: the differ depends only on pkg/schema and the standard library.
func TestDifferImportsAreClean(t *testing.T) {
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
			first := path
			if i := strings.IndexByte(path, '/'); i >= 0 {
				first = path[:i]
			}
			if strings.Contains(first, ".") && path != "github.com/example/price-checker/pkg/schema" {
				t.Errorf("%s imports %q", name, path)
			}
		}
	}
}
