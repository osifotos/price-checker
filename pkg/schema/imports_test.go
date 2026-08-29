package schema_test

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// NFR-5.3: pkg/schema must stay importable by external consumers with a minimal
// dependency footprint. Non-test files may import only the standard library and
// gopkg.in/yaml.v3.
func TestNoDisallowedImports(t *testing.T) {
	allowed := map[string]bool{"gopkg.in/yaml.v3": true}

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
			if isStdlib(path) || allowed[path] {
				continue
			}
			t.Errorf("%s imports disallowed package %q", name, path)
		}
	}
}

func isStdlib(path string) bool {
	first := path
	if i := strings.IndexByte(path, '/'); i >= 0 {
		first = path[:i]
	}
	return !strings.Contains(first, ".")
}
