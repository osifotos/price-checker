package render

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/mtosin123/tf-price_checker/pkg/schema"
)

type tableRenderer struct{}

func (tableRenderer) Render(w io.Writer, b *schema.Breakdown, opts Options) error {
	p := paletteFor(opts.Color)
	showHourly := opts.Period == schema.PeriodHour || opts.ShowComponents

	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	if showHourly {
		fmt.Fprintln(tw, "NAME\tMONTHLY\tHOURLY")
	} else {
		fmt.Fprintln(tw, "NAME\tMONTHLY")
	}
	for _, r := range b.Resources {
		if showHourly {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", r.Address, money2(r.MonthlyCost), rate4(r.HourlyCost))
		} else {
			fmt.Fprintf(tw, "%s\t%s\n", r.Address, money2(r.MonthlyCost))
		}
		if opts.ShowComponents {
			for _, c := range r.CostComponents {
				fmt.Fprintf(tw, "  - %s\t%s\t%s\n", c.Name, money2(c.MonthlyCost), rate4(c.HourlyCost))
			}
		}
	}
	if showHourly {
		fmt.Fprintf(tw, "%sPROJECT TOTAL%s\t%s%s%s\t%s\n", p.bold, p.reset, p.bold, money2(b.TotalMonthly), p.reset, rate4(b.TotalHourly))
	} else {
		fmt.Fprintf(tw, "%sPROJECT TOTAL%s\t%s%s%s\n", p.bold, p.reset, p.bold, money2(b.TotalMonthly), p.reset)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	s := b.Summary
	fmt.Fprintf(w, "\n%d of %d resources estimated", s.ResourcesEstimated, s.ResourcesTotal)
	if s.ComponentsNotEstimated > 0 {
		fmt.Fprintf(w, "; %d component(s) not estimated", s.ComponentsNotEstimated)
	}
	fmt.Fprintln(w)

	if len(b.NotEstimated) > 0 {
		fmt.Fprintf(w, "\n%sNOT ESTIMATED (%d)%s\n", p.warn, len(b.NotEstimated), p.reset)
		net := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
		for _, ne := range b.NotEstimated {
			comp := ne.Component
			if comp == "" {
				comp = "-"
			}
			fmt.Fprintf(net, "%s\t%s\t%s\t%s\n", ne.Address, comp, ne.ReasonCode, ne.Message)
		}
		if err := net.Flush(); err != nil {
			return err
		}
	}
	return nil
}

func (tableRenderer) RenderDiff(w io.Writer, d *schema.DiffResult, opts Options) error {
	p := paletteFor(opts.Color)
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tPRIOR\tNEW\tCHANGE")
	for _, c := range d.Changes {
		note := ""
		switch {
		case c.NotEstimated:
			note = "  (not estimated)"
		case c.Kind == schema.DeltaAdded:
			note = "  (added)"
		case c.Kind == schema.DeltaRemoved:
			note = "  (removed)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s%s\n", c.Address, money2(c.PriorMonthly), money2(c.NewMonthly), signed2(c.DeltaMonthly), note)
	}
	fmt.Fprintf(tw, "%sTOTAL CHANGE%s\t\t\t%s%s / month%s\n", p.bold, p.reset, p.bold, signed2(d.TotalDeltaMonthly), p.reset)
	return tw.Flush()
}
