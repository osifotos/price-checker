package cli

import (
	"context"
	"log/slog"
	"time"

	awspricing "github.com/aws/aws-sdk-go-v2/service/pricing"

	"github.com/osifotos/price-checker/internal/awsauth"
	"github.com/osifotos/price-checker/internal/config"
	"github.com/osifotos/price-checker/internal/engine"
	"github.com/osifotos/price-checker/internal/estimator"
	"github.com/osifotos/price-checker/internal/plan"
	"github.com/osifotos/price-checker/internal/pricing"
	pricingaws "github.com/osifotos/price-checker/internal/pricing/aws"
	"github.com/osifotos/price-checker/internal/render"
	"github.com/osifotos/price-checker/pkg/schema"
)

// buildEstimator constructs the production, AWS-backed estimator.
func buildEstimator(ctx context.Context, cfg config.Config, log *slog.Logger) (Estimator, func(), error) {
	sdkCfg, err := awsauth.Load(ctx, cfg.Profile, cfg.AWSRegion)
	if err != nil {
		return nil, nil, err
	}
	sdkCfg.Region = awsauth.PriceListRegion(sdkCfg)

	raw := pricing.NewPriceListClient(awspricing.NewFromConfig(sdkCfg), pricing.ClientConfig{
		Concurrency: cfg.Concurrency,
		Logger:      log,
	})
	cached, err := pricing.NewCachingClient(raw, pricing.CacheOptions{
		Dir:     cfg.CacheDir,
		TTL:     cfg.CacheTTL,
		NoCache: cfg.NoCache,
		Refresh: cfg.RefreshCache,
	})
	if err != nil {
		return nil, nil, err
	}

	est := estimator.New(estimator.Deps{
		Loader:      plan.NewLoader(),
		Parser:      plan.NewParser(),
		Catalog:     pricingaws.NewCatalog(),
		Querier:     cached,
		Region:      pricing.RegionResolver{},
		Agg:         engine.Aggregator{},
		Log:         log,
		ToolVersion: version,
		Now:         time.Now,
	})
	return est, func() { _ = raw.Close() }, nil
}

func renderOptions(cfg config.Config) render.Options {
	period := schema.PeriodMonth
	if cfg.Period == "hour" {
		period = schema.PeriodHour
	}
	return render.Options{
		Period:         period,
		ShowComponents: cfg.ShowComponents,
		Color:          !cfg.NoColor,
	}
}
