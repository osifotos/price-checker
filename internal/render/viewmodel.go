package render

import (
	"math/big"
	"strings"

	"github.com/example/price-checker/pkg/schema"
)

// ---- money formatting (the only place money is formatted in this unit) ----

func ratOf(s string) *big.Rat {
	r, err := schema.StringToRat(s)
	if err != nil {
		return new(big.Rat)
	}
	return r
}

// money2 formats a decimal string as "$1,234.56".
func money2(s string) string {
	return "$" + group(schema.RatToString(ratOf(s), 2))
}

// rate4 formats a small per-hour rate as "$0.0416".
func rate4(s string) string {
	return "$" + schema.RatToString(ratOf(s), 4)
}

// signed2 formats a delta as "+$12.34", "-$12.34", or "$0.00".
func signed2(s string) string {
	r := ratOf(s)
	switch r.Sign() {
	case 0:
		return "$0.00"
	case 1:
		return "+$" + group(schema.RatToString(r, 2))
	default:
		abs := new(big.Rat).Abs(r)
		return "-$" + group(schema.RatToString(abs, 2))
	}
}

// group inserts thousands separators into the integer part of a plain decimal
// string (which may carry a leading "-").
func group(s string) string {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	var b strings.Builder
	n := len(intPart)
	for i, c := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	out := b.String() + frac
	if neg {
		out = "-" + out
	}
	return out
}

// ---- palette ----

type palette struct{ bold, dim, warn, reset string }

func paletteFor(color bool) palette {
	if !color {
		return palette{}
	}
	return palette{bold: "\x1b[1m", dim: "\x1b[2m", warn: "\x1b[33m", reset: "\x1b[0m"}
}

// ---- HTML view model ----

type htmlVM struct {
	IsDiff        bool
	Title         string
	Currency      string
	TotalMonthly  string
	TotalHourly   string
	DeltaHeadline string
	Modules       []htmlModule
	Resources     []htmlResource
	NotEstimated  []schema.NotEstimated
	Changes       []htmlChange
	Meta          schema.RunMetadata
}

type htmlModule struct {
	Name    string
	Monthly string
	Count   int
}

type htmlResource struct {
	Address    string
	Type       string
	Region     string
	Monthly    string
	Hourly     string
	Components []htmlComponent
}

type htmlComponent struct {
	Name     string
	Unit     string
	Price    string
	Quantity string
	Monthly  string
}

type htmlChange struct {
	Address      string
	Kind         string
	Prior        string
	New          string
	Delta        string
	NotEstimated bool
}

func breakdownVM(b *schema.Breakdown) htmlVM {
	vm := htmlVM{
		Title:        "Cost breakdown",
		Currency:     b.Currency,
		TotalMonthly: money2(b.TotalMonthly),
		TotalHourly:  rate4(b.TotalHourly),
		NotEstimated: b.NotEstimated,
		Meta:         b.Metadata,
	}
	for _, m := range b.Modules {
		name := m.Module
		if name == "" {
			name = "(root)"
		}
		vm.Modules = append(vm.Modules, htmlModule{Name: name, Monthly: money2(m.MonthlyCost), Count: m.ResourceQty})
	}
	for _, r := range b.Resources {
		hr := htmlResource{
			Address: r.Address, Type: r.Type, Region: r.Region,
			Monthly: money2(r.MonthlyCost), Hourly: rate4(r.HourlyCost),
		}
		for _, c := range r.CostComponents {
			hr.Components = append(hr.Components, htmlComponent{
				Name: c.Name, Unit: c.Unit,
				Price: rate4(c.PricePerUnit), Quantity: c.MonthlyQuantity, Monthly: money2(c.MonthlyCost),
			})
		}
		vm.Resources = append(vm.Resources, hr)
	}
	return vm
}

func diffVM(d *schema.DiffResult) htmlVM {
	vm := htmlVM{
		IsDiff:        true,
		Title:         "Cost change",
		Currency:      d.Currency,
		DeltaHeadline: signed2(d.TotalDeltaMonthly) + " / month  (" + money2(d.TotalPriorMonthly) + " → " + money2(d.TotalNewMonthly) + ")",
		Meta:          d.Metadata,
	}
	for _, c := range d.Changes {
		vm.Changes = append(vm.Changes, htmlChange{
			Address: c.Address, Kind: string(c.Kind),
			Prior: money2(c.PriorMonthly), New: money2(c.NewMonthly), Delta: signed2(c.DeltaMonthly),
			NotEstimated: c.NotEstimated,
		})
	}
	return vm
}
