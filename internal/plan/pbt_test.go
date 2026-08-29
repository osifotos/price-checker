package plan

import (
	"encoding/json"
	"testing"

	"pgregory.net/rapid"
)

// PBT-03 / PBT-07: parsing a generated plan preserves the set of managed
// resource addresses and each attribute's known/unknown classification.
func TestParsePreservesAddressesAndKnownness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		type genRes struct {
			addr       string
			typ        string
			action     string
			knownAttr  string
			unknownKey string
		}

		n := rapid.IntRange(0, 8).Draw(t, "n")
		gen := make([]genRes, 0, n)
		seen := map[string]bool{}
		for i := 0; i < n; i++ {
			name := rapid.StringMatching(`[a-z]{1,6}`).Draw(t, "name")
			typ := rapid.SampledFrom([]string{"aws_instance", "aws_s3_bucket", "aws_nat_gateway", "aws_db_instance"}).Draw(t, "typ")
			addr := typ + "." + name
			if seen[addr] {
				continue
			}
			seen[addr] = true
			gen = append(gen, genRes{
				addr:       addr,
				typ:        typ,
				action:     rapid.SampledFrom([]string{"create", "update", "no-op"}).Draw(t, "action"),
				knownAttr:  rapid.SampledFrom([]string{"instance_type", "engine", "bucket"}).Draw(t, "ka"),
				unknownKey: rapid.SampledFrom([]string{"arn", "id"}).Draw(t, "uk"),
			})
		}

		doc := map[string]any{
			"format_version":    "1.2",
			"terraform_version": "1.9.0",
			"resource_changes":  []any{},
		}
		var rcs []any
		for _, g := range gen {
			rcs = append(rcs, map[string]any{
				"address":        g.addr,
				"module_address": "",
				"mode":           "managed",
				"type":           g.typ,
				"name":           g.addr[len(g.typ)+1:],
				"provider_name":  "registry.terraform.io/hashicorp/aws",
				"change": map[string]any{
					"actions":       []any{g.action},
					"before":        nil,
					"after":         map[string]any{g.knownAttr: "known-value"},
					"after_unknown": map[string]any{g.unknownKey: true},
				},
			})
		}
		doc["resource_changes"] = rcs

		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		p, err := NewParser().Parse(raw)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		got := map[string]Resource{}
		for _, r := range p.AllResources() {
			got[r.Address] = r
		}
		if len(got) != len(gen) {
			t.Fatalf("got %d resources, want %d", len(got), len(gen))
		}
		for _, g := range gen {
			r, ok := got[g.addr]
			if !ok {
				t.Fatalf("missing %q", g.addr)
			}
			if _, ok := r.Attributes.String(g.knownAttr); !ok {
				t.Errorf("%s: known attr %q not readable", g.addr, g.knownAttr)
			}
			if r.Attributes.IsKnown(g.unknownKey) {
				t.Errorf("%s: %q should be unknown", g.addr, g.unknownKey)
			}
		}
	})
}
