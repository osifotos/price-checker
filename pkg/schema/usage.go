package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// UsageFile is the parsed form of an Infracost-style usage YAML file. It supplies
// monthly usage quantities for usage-based cost components, keyed by resource
// address.
type UsageFile struct {
	Version       string                   `json:"version" yaml:"version"`
	ResourceUsage map[string]ResourceUsage `json:"resource_usage" yaml:"resource_usage"`
}

// ResourceUsage maps a component usage key (for example
// "monthly_data_processed_gb") to a monthly quantity. Unknown keys are preserved
// here; U3 validates them against the pricing catalog and warns.
type ResourceUsage map[string]float64

// usageFileWire is the exact on-disk shape; used to reject unknown top-level keys.
type usageFileWire struct {
	Version       string                        `yaml:"version"`
	ResourceUsage map[string]map[string]float64 `yaml:"resource_usage"`
}

// ParseUsageFile parses b as a usage YAML document. Unknown top-level keys are an
// error; unknown per-resource component keys are kept for the caller to validate.
func ParseUsageFile(b []byte) (UsageFile, error) {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var w usageFileWire
	if err := dec.Decode(&w); err != nil {
		return UsageFile{}, fmt.Errorf("schema: parse usage file: %w", err)
	}
	return usageFromWire(w.Version, w.ResourceUsage), nil
}

// Marshal renders u as deterministic YAML. yaml.v3 sorts map keys, so resource
// addresses and component keys are both emitted in sorted order. This is the
// round-trip target for PBT-02.
func (u UsageFile) Marshal() ([]byte, error) {
	w := usageFileWire{Version: u.Version, ResourceUsage: map[string]map[string]float64{}}
	for addr, ru := range u.ResourceUsage {
		m := map[string]float64{}
		for k, v := range ru {
			m[k] = v
		}
		w.ResourceUsage[addr] = m
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(w); err != nil {
		return nil, fmt.Errorf("schema: marshal usage file: %w", err)
	}
	_ = enc.Close()
	return buf.Bytes(), nil
}

// MarshalJSON emits u with sorted keys so JSON output is deterministic.
func (u UsageFile) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"version":`)
	vb, _ := json.Marshal(u.Version)
	buf.Write(vb)
	buf.WriteString(`,"resource_usage":{`)
	for i, addr := range sortedUsageKeys(u.ResourceUsage) {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, _ := json.Marshal(addr)
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(marshalResourceUsage(u.ResourceUsage[addr]))
	}
	buf.WriteString("}}")
	return buf.Bytes(), nil
}

// UnmarshalJSON is the inverse of MarshalJSON.
func (u *UsageFile) UnmarshalJSON(b []byte) error {
	var raw struct {
		Version       string                        `json:"version"`
		ResourceUsage map[string]map[string]float64 `json:"resource_usage"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*u = usageFromWire(raw.Version, raw.ResourceUsage)
	return nil
}

func usageFromWire(version string, ru map[string]map[string]float64) UsageFile {
	out := UsageFile{Version: version, ResourceUsage: map[string]ResourceUsage{}}
	for addr, comps := range ru {
		m := ResourceUsage{}
		for k, v := range comps {
			m[k] = v
		}
		out.ResourceUsage[addr] = m
	}
	return out
}

func marshalResourceUsage(ru ResourceUsage) []byte {
	var buf bytes.Buffer
	buf.WriteByte('{')
	keys := make([]string, 0, len(ru))
	for k := range ru {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		buf.Write(kb)
		buf.WriteByte(':')
		vb, _ := json.Marshal(ru[k])
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

func sortedUsageKeys(m map[string]ResourceUsage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
