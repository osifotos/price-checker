package usage

import (
	"fmt"
	"os"
	"sort"

	"github.com/osifotos/price-checker/pkg/schema"
)

// Model exposes usage assumptions to pricers.
type Model interface {
	// For returns the usage assumptions for a resource address. The result is
	// never nil.
	For(address string) schema.ResourceUsage
	// Warnings returns non-fatal validation messages.
	Warnings() []string
	// Used reports whether a non-empty usage file was loaded.
	Used() bool
	// Validate records warnings for addresses in the file that are not in the
	// given set of plan addresses. Safe to call once.
	Validate(planAddresses map[string]bool)
}

// Load reads a usage file. An empty path returns an empty model with no error.
// Malformed YAML is a hard error; unknown component keys produce warnings.
func Load(path string) (Model, error) {
	if path == "" {
		return emptyModel{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read usage file: %w", err)
	}
	uf, err := schema.ParseUsageFile(b)
	if err != nil {
		return nil, err
	}
	m := &fileModel{file: uf}
	for addr, comps := range uf.ResourceUsage {
		for k := range comps {
			if !knownUsageKeys[k] {
				m.warnings = append(m.warnings, fmt.Sprintf("usage file: unknown key %q for %s (ignored)", k, addr))
			}
		}
	}
	sort.Strings(m.warnings)
	return m, nil
}

type fileModel struct {
	file      schema.UsageFile
	warnings  []string
	validated bool
}

func (m *fileModel) For(address string) schema.ResourceUsage {
	if ru, ok := m.file.ResourceUsage[address]; ok {
		return ru
	}
	return schema.ResourceUsage{}
}

func (m *fileModel) Warnings() []string { return m.warnings }

func (m *fileModel) Used() bool { return len(m.file.ResourceUsage) > 0 }

func (m *fileModel) Validate(planAddresses map[string]bool) {
	if m.validated {
		return
	}
	m.validated = true
	var extra []string
	for addr := range m.file.ResourceUsage {
		if !planAddresses[addr] {
			extra = append(extra, fmt.Sprintf("usage file: address %q is not in the plan (ignored)", addr))
		}
	}
	sort.Strings(extra)
	m.warnings = append(m.warnings, extra...)
}

// emptyModel is used when no usage file is supplied.
type emptyModel struct{}

func (emptyModel) For(string) schema.ResourceUsage { return schema.ResourceUsage{} }
func (emptyModel) Warnings() []string              { return nil }
func (emptyModel) Used() bool                      { return false }
func (emptyModel) Validate(map[string]bool)        {}
