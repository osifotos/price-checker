package plan

import "sort"

// AttrMap is a read-only view over a resource's planned attributes plus the
// parallel "unknown" tree from after_unknown. All getters return a second bool
// that is true when the value is present AND known (not "known after apply").
type AttrMap struct {
	values  map[string]any
	unknown map[string]any // mirrors values; a true (or nested true) means unknown
}

func newAttrMap(values, unknown map[string]any) AttrMap {
	if values == nil {
		values = map[string]any{}
	}
	return AttrMap{values: values, unknown: unknown}
}

func (a AttrMap) isEmpty() bool { return len(a.values) == 0 }

// NewAttrMapForTest builds an AttrMap directly from decoded attribute values and
// an after_unknown tree. It lets other packages' tests construct a plan.Resource
// without assembling a whole plan document.
func NewAttrMapForTest(values, unknown map[string]any) AttrMap {
	return newAttrMap(values, unknown)
}

// Has reports whether key is present in the planned attributes.
func (a AttrMap) Has(key string) bool {
	_, ok := a.values[key]
	return ok
}

// IsKnown reports whether key's value is known at plan time.
func (a AttrMap) IsKnown(key string) bool {
	if a.unknown == nil {
		return true
	}
	return !isUnknownNode(a.unknown[key])
}

// String returns key as a string. ok is false if absent, unknown, or not a string.
func (a AttrMap) String(key string) (string, bool) {
	if !a.IsKnown(key) {
		return "", false
	}
	s, ok := a.values[key].(string)
	return s, ok
}

// Float returns key as a float64. JSON numbers decode as float64.
func (a AttrMap) Float(key string) (float64, bool) {
	if !a.IsKnown(key) {
		return 0, false
	}
	f, ok := a.values[key].(float64)
	return f, ok
}

// Int returns key as an int64 (truncating any fractional part).
func (a AttrMap) Int(key string) (int64, bool) {
	f, ok := a.Float(key)
	if !ok {
		return 0, false
	}
	return int64(f), true
}

// Bool returns key as a bool.
func (a AttrMap) Bool(key string) (bool, bool) {
	if !a.IsKnown(key) {
		return false, false
	}
	b, ok := a.values[key].(bool)
	return b, ok
}

// StringSlice returns key as a []string. ok is false unless every element is a
// known string.
func (a AttrMap) StringSlice(key string) ([]string, bool) {
	if !a.IsKnown(key) {
		return nil, false
	}
	raw, ok := a.values[key].([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(raw))
	for _, e := range raw {
		s, ok := e.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

// Block returns the first element of a nested block list (for example
// root_block_device) as an AttrMap. ok is false if absent, not a list of
// objects, or empty.
func (a AttrMap) Block(key string) (AttrMap, bool) {
	blocks := a.Blocks(key)
	if len(blocks) == 0 {
		return AttrMap{}, false
	}
	return blocks[0], true
}

// Blocks returns every element of a nested block list as AttrMaps. Non-object
// elements are skipped.
func (a AttrMap) Blocks(key string) []AttrMap {
	raw, ok := a.values[key].([]any)
	if !ok {
		return nil
	}
	var unkList []any
	if a.unknown != nil {
		unkList, _ = a.unknown[key].([]any)
	}
	out := make([]AttrMap, 0, len(raw))
	for i, e := range raw {
		obj, ok := e.(map[string]any)
		if !ok {
			continue
		}
		var unkObj map[string]any
		if i < len(unkList) {
			unkObj, _ = unkList[i].(map[string]any)
		}
		out = append(out, newAttrMap(obj, unkObj))
	}
	return out
}

// Keys returns the present attribute keys, sorted.
func (a AttrMap) Keys() []string {
	out := make([]string, 0, len(a.values))
	for k := range a.values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// isUnknownNode reports whether an after_unknown node marks its value unknown.
// A bare true means "unknown". A nested object/array means "partially known";
// for the purposes of a top-level getter that is treated as known (callers
// descend with Block/Blocks to see the nested unknowns).
func isUnknownNode(n any) bool {
	b, ok := n.(bool)
	return ok && b
}
