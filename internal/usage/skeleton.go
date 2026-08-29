package usage

import (
	"bufio"
	"fmt"
	"io"
	"sort"

	"github.com/example/price-checker/internal/plan"
)

// GenerateSkeleton writes a starter usage YAML to w listing every usage-based
// component of every in-plan resource, with zero values and an inline comment
// describing each unit. Resources with no usage-based component are omitted.
//
// The output is valid input to Load: running with the generated file (values
// still zero) yields the same result as running with no usage file.
func GenerateSkeleton(p *plan.Plan, w io.Writer) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	type entry struct {
		address string
		keys    []usageKey
	}
	var entries []entry
	for _, r := range p.Resources() {
		if ks, ok := keysByType[r.Type]; ok && len(ks) > 0 {
			entries = append(entries, entry{address: r.Address, keys: ks})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].address < entries[j].address })

	fmt.Fprintln(bw, `version: "0.1"`)
	if len(entries) == 0 {
		fmt.Fprintln(bw, "resource_usage: {}")
		return bw.Flush()
	}
	fmt.Fprintln(bw, "resource_usage:")
	for _, e := range entries {
		fmt.Fprintf(bw, "  %s:\n", yamlKey(e.address))
		for _, k := range e.keys {
			fmt.Fprintf(bw, "    %s: 0  # %s\n", k.Key, k.Description)
		}
	}
	return bw.Flush()
}

// yamlKey quotes an address if it contains characters that would confuse a YAML
// parser (brackets from count/for_each indices, quotes).
func yamlKey(s string) string {
	for _, r := range s {
		if r == '[' || r == ']' || r == '"' || r == '\'' || r == ':' || r == '#' {
			return fmt.Sprintf("%q", s)
		}
	}
	return s
}
