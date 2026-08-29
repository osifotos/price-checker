package render

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// NFR-5.3: renderers depend only on pkg/schema and the standard library.
func TestRendererImportsAreClean(t *testing.T) {
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
			ok := isStd(path) || path == "github.com/example/price-checker/pkg/schema"
			if !ok {
				t.Errorf("%s imports %q — renderers may only use stdlib + pkg/schema", name, path)
			}
		}
	}
}

func isStd(path string) bool {
	first := path
	if i := strings.IndexByte(path, '/'); i >= 0 {
		first = path[:i]
	}
	return !strings.Contains(first, ".")
}
