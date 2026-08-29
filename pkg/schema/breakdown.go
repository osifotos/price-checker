package schema

import "sort"

// Breakdown is the root document produced by `price-checker breakdown --format json`.
// All monetary fields are decimal strings (see package docs). Slices are emitted
// as [] never null; call Canonicalize before encoding to get deterministic order.
type Breakdown struct {
	SchemaVersion string          `json:"schema_version"`
	Currency      string          `json:"currency"`
	Period        Period          `json:"period"`
	Resources     []Resource      `json:"resources"`
	Modules       []ModuleTotal   `json:"modules"`
	TotalMonthly  string          `json:"total_monthly"`
	TotalHourly   string          `json:"total_hourly"`
	NotEstimated  []NotEstimated  `json:"not_estimated"`
	Summary       CoverageSummary `json:"summary"`
	Metadata      RunMetadata     `json:"metadata"`
}

// Resource is one planned resource and its priced components.
type Resource struct {
	Address        string          `json:"address"`
	Type           string          `json:"type"`
	Name           string          `json:"name"`
	Module         string          `json:"module"`
	Region         string          `json:"region"`
	MonthlyCost    string          `json:"monthly_cost"`
	HourlyCost     string          `json:"hourly_cost"`
	CostComponents []CostComponent `json:"cost_components"`
}

// CostComponent is a single priced line item within a resource.
type CostComponent struct {
	Name            string `json:"name"`
	Unit            string `json:"unit"`
	PricePerUnit    string `json:"price_per_unit"`
	MonthlyQuantity string `json:"monthly_quantity"`
	MonthlyCost     string `json:"monthly_cost"`
	HourlyCost      string `json:"hourly_cost"`
	UsageBased      bool   `json:"usage_based"`
}

// ModuleTotal is the cost rollup for one Terraform module path ("" = root).
type ModuleTotal struct {
	Module      string `json:"module"`
	MonthlyCost string `json:"monthly_cost"`
	HourlyCost  string `json:"hourly_cost"`
	ResourceQty int    `json:"resource_count"`
}

// NotEstimated records a resource or component that received no cost estimate.
type NotEstimated struct {
	Address    string     `json:"address"`
	Type       string     `json:"type"`
	Component  string     `json:"component,omitempty"`
	ReasonCode ReasonCode `json:"reason_code"`
	Message    string     `json:"message"`
}

// CoverageSummary counts what was and was not estimated.
type CoverageSummary struct {
	ResourcesTotal         int `json:"resources_total"`
	ResourcesEstimated     int `json:"resources_estimated"`
	ResourcesNotEstimated  int `json:"resources_not_estimated"`
	ComponentsEstimated    int `json:"components_estimated"`
	ComponentsNotEstimated int `json:"components_not_estimated"`
}

// RunMetadata describes how a document was produced. GeneratedAt is supplied by
// the caller (RFC3339 UTC); this package never reads the clock.
type RunMetadata struct {
	ToolVersion       string   `json:"tool_version"`
	GeneratedAt       string   `json:"generated_at"`
	PlanFormatVersion string   `json:"plan_format_version"`
	Regions           []string `json:"regions"`
	UsageFileUsed     bool     `json:"usage_file_used"`
	PriceQueries      int      `json:"price_queries"`
	CacheHitRatio     float64  `json:"cache_hit_ratio"`
	DedupHits         int      `json:"dedup_hits"`
}

// NewBreakdown returns a Breakdown with the current SchemaVersion, USD currency,
// and non-nil (empty) slices.
func NewBreakdown(period Period) *Breakdown {
	if !period.Valid() {
		period = PeriodMonth
	}
	return &Breakdown{
		SchemaVersion: SchemaVersion,
		Currency:      "USD",
		Period:        period,
		Resources:     []Resource{},
		Modules:       []ModuleTotal{},
		NotEstimated:  []NotEstimated{},
		TotalMonthly:  "0",
		TotalHourly:   "0",
	}
}

// Canonicalize sorts every slice in b by its documented key so that encoding is
// deterministic. Producers call this once before marshalling.
func (b *Breakdown) Canonicalize() {
	if b.Resources == nil {
		b.Resources = []Resource{}
	}
	if b.Modules == nil {
		b.Modules = []ModuleTotal{}
	}
	if b.NotEstimated == nil {
		b.NotEstimated = []NotEstimated{}
	}
	if b.Metadata.Regions == nil {
		b.Metadata.Regions = []string{}
	}
	for i := range b.Resources {
		if b.Resources[i].CostComponents == nil {
			b.Resources[i].CostComponents = []CostComponent{}
		}
	}
	sort.Slice(b.Resources, func(i, j int) bool { return b.Resources[i].Address < b.Resources[j].Address })
	for i := range b.Resources {
		cc := b.Resources[i].CostComponents
		sort.Slice(cc, func(x, y int) bool { return cc[x].Name < cc[y].Name })
	}
	sort.Slice(b.Modules, func(i, j int) bool { return b.Modules[i].Module < b.Modules[j].Module })
	sort.Slice(b.NotEstimated, func(i, j int) bool {
		if b.NotEstimated[i].Address != b.NotEstimated[j].Address {
			return b.NotEstimated[i].Address < b.NotEstimated[j].Address
		}
		return b.NotEstimated[i].Component < b.NotEstimated[j].Component
	})
	sort.Strings(b.Metadata.Regions)
}
