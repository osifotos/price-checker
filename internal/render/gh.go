package render

import (
	"bufio"
	"fmt"
	"io"

	"github.com/example/price-checker/pkg/schema"
)

type ghRenderer struct{}

func (ghRenderer) Render(w io.Writer, b *schema.Breakdown, _ Options) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	fmt.Fprintf(bw, "### price-checker — estimated monthly cost: **%s**\n\n", money2(b.TotalMonthly))

	fmt.Fprintln(bw, "| Resource | Monthly |")
	fmt.Fprintln(bw, "|---|--:|")
	for _, r := range b.Resources {
		fmt.Fprintf(bw, "| `%s` | %s |\n", r.Address, money2(r.MonthlyCost))
	}
	fmt.Fprintln(bw)

	if b.Summary.ResourcesNotEstimated > 0 || b.Summary.ComponentsNotEstimated > 0 {
		fmt.Fprintf(bw, "> :warning: %d resource(s) and %d component(s) not estimated\n\n",
			b.Summary.ResourcesNotEstimated, b.Summary.ComponentsNotEstimated)
	}

	fmt.Fprint(bw, "<details><summary>Full breakdown</summary>\n\n")
	fmt.Fprintln(bw, "| Resource | Component | Monthly | Hourly |")
	fmt.Fprintln(bw, "|---|---|--:|--:|")
	for _, r := range b.Resources {
		for _, c := range r.CostComponents {
			fmt.Fprintf(bw, "| `%s` | %s | %s | %s |\n", r.Address, c.Name, money2(c.MonthlyCost), rate4(c.HourlyCost))
		}
	}
	if len(b.NotEstimated) > 0 {
		fmt.Fprint(bw, "\n**Not estimated**\n\n")
		fmt.Fprintln(bw, "| Resource | Component | Reason |")
		fmt.Fprintln(bw, "|---|---|---|")
		for _, ne := range b.NotEstimated {
			comp := ne.Component
			if comp == "" {
				comp = "—"
			}
			fmt.Fprintf(bw, "| `%s` | %s | `%s` |\n", ne.Address, comp, ne.ReasonCode)
		}
	}
	fmt.Fprintln(bw, "\n</details>")
	fmt.Fprintf(bw, "\n<sub>price-checker %s · %s</sub>\n", nonEmpty(b.Metadata.ToolVersion), b.Metadata.GeneratedAt)
	return bw.Flush()
}

func (ghRenderer) RenderDiff(w io.Writer, d *schema.DiffResult, _ Options) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	fmt.Fprintf(bw, "### price-checker — estimated monthly cost change: **%s** (%s → %s)\n\n",
		signed2(d.TotalDeltaMonthly), money2(d.TotalPriorMonthly), money2(d.TotalNewMonthly))

	fmt.Fprintln(bw, "| Resource | Prior | New | Change |")
	fmt.Fprintln(bw, "|---|--:|--:|--:|")
	notEstimated := 0
	for _, c := range d.Changes {
		if c.NotEstimated {
			notEstimated++
			fmt.Fprintf(bw, "| `%s` | — | — | not estimated |\n", c.Address)
			continue
		}
		if c.Kind == schema.DeltaUnchanged {
			continue
		}
		fmt.Fprintf(bw, "| `%s` | %s | %s | **%s** |\n", c.Address, money2(c.PriorMonthly), money2(c.NewMonthly), signed2(c.DeltaMonthly))
	}
	fmt.Fprintln(bw)
	if notEstimated > 0 {
		fmt.Fprintf(bw, "> :warning: %d changed resource(s) not estimated\n\n", notEstimated)
	}
	fmt.Fprintf(bw, "<sub>price-checker %s · %s</sub>\n", nonEmpty(d.Metadata.ToolVersion), d.Metadata.GeneratedAt)
	return bw.Flush()
}

func nonEmpty(s string) string {
	if s == "" {
		return "(dev)"
	}
	return s
}
