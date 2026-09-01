package schema_test

import (
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/mtosin123/tf-price_checker/pkg/schema"
	"pgregory.net/rapid"
)

// PBT-02: Breakdown survives a JSON marshal/unmarshal round trip.
func TestBreakdownJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orig := drawBreakdown(t)
		orig.Canonicalize()

		b, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var back schema.Breakdown
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		back.Canonicalize()

		b2, _ := json.Marshal(&back)
		if string(b) != string(b2) {
			t.Fatalf("round trip not stable:\n a=%s\n b=%s", b, b2)
		}
	})
}

// PBT-02: DiffResult survives a JSON round trip.
func TestDiffJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orig := drawDiff(t)
		orig.Canonicalize()
		b, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var back schema.DiffResult
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		back.Canonicalize()
		b2, _ := json.Marshal(&back)
		if string(b) != string(b2) {
			t.Fatalf("round trip not stable:\n a=%s\n b=%s", b, b2)
		}
	})
}

// PBT-02: UsageFile survives YAML and JSON round trips.
func TestUsageFileRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orig := drawUsage(t)

		y, err := orig.Marshal()
		if err != nil {
			t.Fatalf("yaml marshal: %v", err)
		}
		fromYAML, err := schema.ParseUsageFile(y)
		if err != nil {
			t.Fatalf("yaml parse: %v", err)
		}
		if !reflect.DeepEqual(normalizeUsage(orig), normalizeUsage(fromYAML)) {
			t.Fatalf("yaml round trip changed value:\n in=%#v\n out=%#v\n yaml=%s", orig, fromYAML, y)
		}

		j, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("json marshal: %v", err)
		}
		var fromJSON schema.UsageFile
		if err := json.Unmarshal(j, &fromJSON); err != nil {
			t.Fatalf("json unmarshal: %v", err)
		}
		if !reflect.DeepEqual(normalizeUsage(orig), normalizeUsage(fromJSON)) {
			t.Fatalf("json round trip changed value:\n in=%#v\n out=%#v", orig, fromJSON)
		}
	})
}

func normalizeUsage(u schema.UsageFile) schema.UsageFile {
	if u.ResourceUsage == nil {
		u.ResourceUsage = map[string]schema.ResourceUsage{}
	}
	return u
}

// ---- generators (PBT-07: domain-shaped) ----

func rat64(num, den int64) *big.Rat { return new(big.Rat).SetFrac64(num, den) }

func typeOf(addr string) string {
	for _, p := range strings.Split(addr, ".") {
		if strings.HasPrefix(p, "aws_") {
			return p
		}
	}
	return "aws_instance"
}

var identSampler = rapid.SampledFrom([]string{"web", "db", "assets", "net", "this", "main", "0", "1"})

func drawAddress(t *rapid.T, label string) string {
	head := rapid.SampledFrom([]string{
		"aws_instance", "aws_ebs_volume", "aws_db_instance", "aws_s3_bucket",
		"aws_lambda_function", "aws_lb", "aws_nat_gateway", "aws_eip",
	}).Draw(t, label+"_type")
	name := identSampler.Draw(t, label+"_name")
	if rapid.Bool().Draw(t, label+"_mod") {
		return "module." + identSampler.Draw(t, label+"_modname") + "." + head + "." + name
	}
	return head + "." + name
}

func drawDecimal(t *rapid.T, label string) string {
	n := rapid.Int64Range(0, 5_000_000).Draw(t, label)
	return schema.RatToString(rat64(n, 10000), 4)
}

func drawBreakdown(t *rapid.T) *schema.Breakdown {
	b := schema.NewBreakdown(rapid.SampledFrom([]schema.Period{schema.PeriodMonth, schema.PeriodHour}).Draw(t, "period"))
	n := rapid.IntRange(0, 6).Draw(t, "nres")
	for i := 0; i < n; i++ {
		addr := drawAddress(t, "res")
		ncomp := rapid.IntRange(0, 3).Draw(t, "ncomp")
		var comps []schema.CostComponent
		for j := 0; j < ncomp; j++ {
			comps = append(comps, schema.CostComponent{
				Name:            rapid.SampledFrom([]string{"Instance usage", "Storage", "Requests", "Data processed"}).Draw(t, "cname"),
				Unit:            rapid.SampledFrom([]string{"hours", "GB-months", "requests"}).Draw(t, "unit"),
				PricePerUnit:    drawDecimal(t, "ppu"),
				MonthlyQuantity: drawDecimal(t, "qty"),
				MonthlyCost:     drawDecimal(t, "mc"),
				HourlyCost:      drawDecimal(t, "hc"),
				UsageBased:      rapid.Bool().Draw(t, "ub"),
			})
		}
		b.Resources = append(b.Resources, schema.Resource{
			Address: addr, Type: typeOf(addr), Name: "x", Module: "", Region: "us-east-1",
			MonthlyCost: drawDecimal(t, "rmc"), HourlyCost: drawDecimal(t, "rhc"),
			CostComponents: comps,
		})
	}
	b.TotalMonthly = drawDecimal(t, "tm")
	b.TotalHourly = drawDecimal(t, "th")
	b.Metadata = schema.RunMetadata{
		ToolVersion: "t", GeneratedAt: "2026-08-29T00:00:00Z", PlanFormatVersion: "1.2",
		Regions: []string{"us-east-1"},
	}
	return b
}

func drawDiff(t *rapid.T) *schema.DiffResult {
	d := schema.NewDiffResult(schema.PeriodMonth)
	n := rapid.IntRange(0, 6).Draw(t, "nchg")
	for i := 0; i < n; i++ {
		d.Changes = append(d.Changes, schema.ResourceDelta{
			Address: drawAddress(t, "chg"), Type: "aws_instance",
			Kind:         rapid.SampledFrom([]schema.DeltaKind{schema.DeltaAdded, schema.DeltaRemoved, schema.DeltaChanged, schema.DeltaUnchanged}).Draw(t, "kind"),
			PriorMonthly: drawDecimal(t, "pm"), NewMonthly: drawDecimal(t, "nm"),
			DeltaMonthly: drawDecimal(t, "dm"), DeltaHourly: drawDecimal(t, "dh"),
			NotEstimated: rapid.Bool().Draw(t, "ne"),
		})
	}
	d.Metadata = schema.RunMetadata{ToolVersion: "t", GeneratedAt: "2026-08-29T00:00:00Z", PlanFormatVersion: "1.2", Regions: []string{"us-east-1"}}
	return d
}

func drawUsage(t *rapid.T) schema.UsageFile {
	u := schema.UsageFile{Version: "0.1", ResourceUsage: map[string]schema.ResourceUsage{}}
	n := rapid.IntRange(0, 5).Draw(t, "nu")
	for i := 0; i < n; i++ {
		ru := schema.ResourceUsage{}
		nk := rapid.IntRange(0, 4).Draw(t, "nk")
		for j := 0; j < nk; j++ {
			k := rapid.SampledFrom([]string{"storage_gb", "monthly_requests", "monthly_data_processed_gb", "monthly_tier1_requests"}).Draw(t, "k")
			ru[k] = float64(rapid.IntRange(0, 1_000_000).Draw(t, "v"))
		}
		u.ResourceUsage[drawAddress(t, "u")] = ru
	}
	return u
}
