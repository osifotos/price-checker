// Package estimator orchestrates the plan -> Breakdown pipeline: load, parse,
// resolve regions, price each resource (bounded fan-out), merge usage
// assumptions, and aggregate.
package estimator

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/mtosin123/tf-price_checker/internal/engine"
	"github.com/mtosin123/tf-price_checker/internal/plan"
	"github.com/mtosin123/tf-price_checker/internal/pricing"
	"github.com/mtosin123/tf-price_checker/internal/usage"
	"github.com/mtosin123/tf-price_checker/pkg/schema"
	"golang.org/x/sync/errgroup"
)

// Deps are the collaborators an Estimator needs. The caller (the CLI layer)
// builds the cache-wrapped Querier.
type Deps struct {
	Loader      plan.PlanLoader
	Parser      plan.PlanParser
	Catalog     pricing.Catalog
	Querier     pricing.PriceQuerier
	Region      pricing.RegionResolver
	Agg         engine.Aggregator
	Log         *slog.Logger
	ToolVersion string
	Now         func() time.Time
}

// EstimateRequest is one estimation.
type EstimateRequest struct {
	Source         string
	Stdin          io.Reader
	RegionOverride string
	UsagePath      string
	Concurrency    int
	Period         schema.Period
}

// Estimator turns a plan source into a schema.Breakdown.
type Estimator struct {
	d Deps
}

// New builds an Estimator, applying defaults for optional deps.
func New(d Deps) *Estimator {
	if d.Log == nil {
		d.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Estimator{d: d}
}

// Estimate runs the full pipeline for the planned state.
func (e *Estimator) Estimate(ctx context.Context, req EstimateRequest) (*schema.Breakdown, error) {
	pl, um, err := e.loadPlanAndUsage(ctx, req)
	if err != nil {
		return nil, err
	}
	priced, err := e.priceAll(ctx, pl, pl.Resources(), um, req, false)
	if err != nil {
		return nil, err
	}
	return e.d.Agg.Build(priced, e.metadata(pl, um), period(req.Period)), nil
}

// EstimateBothStates prices the prior and planned states from a single plan,
// reusing the querier so the second pass runs against a warm cache.
func (e *Estimator) EstimateBothStates(ctx context.Context, req EstimateRequest) (prior, planned *schema.Breakdown, err error) {
	pl, um, err := e.loadPlanAndUsage(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	priorPriced, err := e.priceAll(ctx, pl, pl.PriorResources(), um, req, true)
	if err != nil {
		return nil, nil, err
	}
	plannedPriced, err := e.priceAll(ctx, pl, pl.Resources(), um, req, false)
	if err != nil {
		return nil, nil, err
	}
	meta := e.metadata(pl, um)
	return e.d.Agg.Build(priorPriced, meta, period(req.Period)),
		e.d.Agg.Build(plannedPriced, meta, period(req.Period)), nil
}

func (e *Estimator) loadPlanAndUsage(ctx context.Context, req EstimateRequest) (*plan.Plan, usage.Model, error) {
	raw, _, err := e.d.Loader.Load(ctx, req.Source, req.Stdin)
	if err != nil {
		return nil, nil, err
	}
	pl, err := e.d.Parser.Parse(raw)
	if err != nil {
		return nil, nil, err
	}
	um, err := usage.Load(req.UsagePath)
	if err != nil {
		return nil, nil, err
	}
	addrs := map[string]bool{}
	for _, r := range pl.AllResources() {
		addrs[r.Address] = true
	}
	um.Validate(addrs)
	for _, w := range um.Warnings() {
		e.d.Log.Warn(w)
	}
	return pl, um, nil
}

func (e *Estimator) priceAll(ctx context.Context, pl *plan.Plan, resources []plan.Resource, um usage.Model, req EstimateRequest, prior bool) ([]engine.PricedResource, error) {
	results := make([]engine.PricedResource, len(resources))

	limit := req.Concurrency
	if limit <= 0 {
		limit = 8
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)

	for i := range resources {
		i := i
		res := resources[i]
		if prior {
			res = withAttributes(res, res.PriorAttributes)
		}
		g.Go(func() error {
			results[i] = e.priceOne(gctx, pl, res, um, req.RegionOverride)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func (e *Estimator) priceOne(ctx context.Context, pl *plan.Plan, res plan.Resource, um usage.Model, regionOverride string) engine.PricedResource {
	region, ok := e.d.Region.Resolve(res, pl.ProviderConfigs, regionOverride)
	if !ok {
		return engine.PricedResource{Resource: res, NotEstimated: []schema.NotEstimated{{
			Address: res.Address, Type: res.Type,
			ReasonCode: schema.ReasonMissingAttribute, Message: "could not determine the AWS region; set --aws-region",
		}}}
	}

	pr, ok := e.d.Catalog.For(res.Type)
	if !ok {
		return engine.PricedResource{Resource: res, Region: region, NotEstimated: []schema.NotEstimated{{
			Address: res.Address, Type: res.Type,
			ReasonCode: schema.ReasonUnsupportedType, Message: schema.ReasonUnsupportedType.Message(),
		}}}
	}

	out, err := pr.Price(ctx, pricing.PricingInput{
		Resource: res, Usage: um.For(res.Address), Region: region, Q: e.d.Querier,
	})
	if err != nil {
		e.d.Log.Debug("pricer error", "address", res.Address, "err", err)
		return engine.PricedResource{Resource: res, Region: region, NotEstimated: []schema.NotEstimated{{
			Address: res.Address, Type: res.Type,
			ReasonCode: schema.ReasonPricingAPIError, Message: err.Error(),
		}}}
	}
	return engine.PricedResource{
		Resource: res, Region: region,
		Components: out.Components, NotEstimated: out.NotEstimated,
	}
}

func (e *Estimator) metadata(pl *plan.Plan, um usage.Model) schema.RunMetadata {
	m := schema.RunMetadata{
		ToolVersion:       e.d.ToolVersion,
		GeneratedAt:       e.d.Now().UTC().Format(time.RFC3339),
		PlanFormatVersion: pl.FormatVersion,
		UsageFileUsed:     um.Used(),
	}
	if s, ok := e.d.Querier.(interface{ Stats() pricing.QueryStats }); ok {
		st := s.Stats()
		m.PriceQueries = st.Issued
		m.CacheHitRatio = st.CacheHitRatio()
		m.DedupHits = st.DedupHits
	}
	return m
}

func period(p schema.Period) schema.Period {
	if p.Valid() {
		return p
	}
	return schema.PeriodMonth
}

func withAttributes(res plan.Resource, attrs plan.AttrMap) plan.Resource {
	res.Attributes = attrs
	return res
}
