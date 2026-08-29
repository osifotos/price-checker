package schema

import (
	"math/big"
	"sort"
)

// DeltaKind classifies how a resource changed between two cost states.
type DeltaKind string

const (
	DeltaAdded     DeltaKind = "added"
	DeltaRemoved   DeltaKind = "removed"
	DeltaChanged   DeltaKind = "changed"
	DeltaUnchanged DeltaKind = "unchanged"
)

// Valid reports whether k is a recognised delta kind.
func (k DeltaKind) Valid() bool {
	switch k {
	case DeltaAdded, DeltaRemoved, DeltaChanged, DeltaUnchanged:
		return true
	default:
		return false
	}
}

// DiffResult is the root document produced by `price-checker diff --format json`.
type DiffResult struct {
	SchemaVersion     string          `json:"schema_version"`
	Currency          string          `json:"currency"`
	Period            Period          `json:"period"`
	Changes           []ResourceDelta `json:"changes"`
	TotalPriorMonthly string          `json:"total_prior_monthly"`
	TotalNewMonthly   string          `json:"total_new_monthly"`
	TotalDeltaMonthly string          `json:"total_delta_monthly"`
	TotalDeltaHourly  string          `json:"total_delta_hourly"`
	Summary           CoverageSummary `json:"summary"`
	Metadata          RunMetadata     `json:"metadata"`
}

// ResourceDelta is the cost change for one resource address.
type ResourceDelta struct {
	Address      string    `json:"address"`
	Type         string    `json:"type"`
	Kind         DeltaKind `json:"kind"`
	PriorMonthly string    `json:"prior_monthly"`
	NewMonthly   string    `json:"new_monthly"`
	DeltaMonthly string    `json:"delta_monthly"`
	DeltaHourly  string    `json:"delta_hourly"`
	NotEstimated bool      `json:"not_estimated"`
}

// NewDiffResult returns a DiffResult with the current SchemaVersion, USD
// currency, and non-nil (empty) Changes.
func NewDiffResult(period Period) *DiffResult {
	if !period.Valid() {
		period = PeriodMonth
	}
	return &DiffResult{
		SchemaVersion:     SchemaVersion,
		Currency:          "USD",
		Period:            period,
		Changes:           []ResourceDelta{},
		TotalPriorMonthly: "0",
		TotalNewMonthly:   "0",
		TotalDeltaMonthly: "0",
		TotalDeltaHourly:  "0",
	}
}

// Canonicalize sorts Changes by (descending absolute monthly delta, then
// address) so the most significant change is first, deterministically.
func (d *DiffResult) Canonicalize() {
	if d.Changes == nil {
		d.Changes = []ResourceDelta{}
	}
	if d.Metadata.Regions == nil {
		d.Metadata.Regions = []string{}
	}
	sort.SliceStable(d.Changes, func(i, j int) bool {
		ai := absDeltaRat(d.Changes[i].DeltaMonthly)
		aj := absDeltaRat(d.Changes[j].DeltaMonthly)
		if c := ai.Cmp(aj); c != 0 {
			return c > 0
		}
		return d.Changes[i].Address < d.Changes[j].Address
	})
	sort.Strings(d.Metadata.Regions)
}

// absDeltaRat parses a delta string to its absolute value, treating an
// unparseable value as zero (Canonicalize must never fail).
func absDeltaRat(s string) *big.Rat {
	r, err := StringToRat(s)
	if err != nil {
		return new(big.Rat)
	}
	return r.Abs(r)
}
