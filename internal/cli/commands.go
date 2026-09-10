package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awspricing "github.com/aws/aws-sdk-go-v2/service/pricing"
	pricingtypes "github.com/aws/aws-sdk-go-v2/service/pricing/types"
	"github.com/urfave/cli/v2"

	"github.com/osifotos/price-checker/internal/awsauth"
	"github.com/osifotos/price-checker/internal/config"
	"github.com/osifotos/price-checker/internal/diff"
	"github.com/osifotos/price-checker/internal/estimator"
	"github.com/osifotos/price-checker/internal/render"
	"github.com/osifotos/price-checker/internal/usage"
	"github.com/osifotos/price-checker/pkg/schema"
)

func (e *env) versionAction(c *cli.Context) error {
	fmt.Fprintln(e.stdout, versionString())
	return nil
}

func (e *env) breakdownAction(c *cli.Context) error {
	cfg, prov, warns, err := resolveConfig(c, e.noColor())
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel, e.stderr)
	logWarnings(log, warns)
	logProvenance(log, cfg, prov)

	r, _, err := render.RendererFor(cfg.Format)
	if err != nil {
		return err
	}

	est, cleanup, err := e.deps.NewEstimator(c.Context, cfg, log)
	if err != nil {
		return err
	}
	defer cleanup()

	b, err := est.Estimate(c.Context, estimator.EstimateRequest{
		Source:         cfg.Path,
		Stdin:          e.stdin,
		RegionOverride: cfg.AWSRegion,
		UsagePath:      cfg.UsageFile,
		Concurrency:    cfg.Concurrency,
		Period:         renderOptions(cfg).Period,
	})
	if err != nil {
		return err
	}

	if err := writeRendered(e, cfg, func(w io.Writer) error { return r.Render(w, b, renderOptions(cfg)) }); err != nil {
		return err
	}

	total, _ := strconv.ParseFloat(b.TotalMonthly, 64)
	return exitFor(Outcome{
		NotEstimatedCount: b.Summary.ResourcesNotEstimated + b.Summary.ComponentsNotEstimated,
		MonthlyTotal:      total,
	}, cfg)
}

func (e *env) diffAction(c *cli.Context) error {
	cfg, _, warns, err := resolveConfig(c, e.noColor())
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel, e.stderr)
	logWarnings(log, warns)

	_, dr, err := render.RendererFor(cfg.Format)
	if err != nil {
		return err
	}

	est, cleanup, err := e.deps.NewEstimator(c.Context, cfg, log)
	if err != nil {
		return err
	}
	defer cleanup()

	var base, proposed *schema.Breakdown

	if cfg.FromPlan {
		base, proposed, err = est.EstimateBothStates(c.Context, e.request(cfg))
		if err != nil {
			return err
		}
	} else {
		if cfg.CompareTo == "" {
			return fmt.Errorf("diff needs --compare-to <baseline> or --from-plan")
		}
		proposed, err = e.estimateOrLoad(c.Context, est, cfg, cfg.Path)
		if err != nil {
			return err
		}
		base, err = e.estimateOrLoad(c.Context, est, cfg, cfg.CompareTo)
		if err != nil {
			return err
		}
	}

	d := diff.Differ{}.Diff(base, proposed)

	if err := writeRendered(e, cfg, func(w io.Writer) error { return dr.RenderDiff(w, d, renderOptions(cfg)) }); err != nil {
		return err
	}

	delta, _ := strconv.ParseFloat(d.TotalDeltaMonthly, 64)
	notEst := d.Summary.ResourcesNotEstimated
	return exitFor(Outcome{NotEstimatedCount: notEst, MonthlyDelta: &delta}, cfg)
}

func (e *env) reportAction(c *cli.Context) error {
	if !c.IsSet("format") {
		_ = c.Set("format", "html")
	}
	if !c.IsSet("out") {
		_ = c.Set("out", "price-checker-report.html")
	}
	if c.IsSet("compare-to") || c.Bool("from-plan") {
		return e.diffAction(c)
	}
	return e.breakdownAction(c)
}

func (e *env) usageGenerateAction(c *cli.Context) error {
	cfg, _, _, err := resolveConfig(c, e.noColor())
	if err != nil {
		return err
	}
	raw, _, err := e.deps.Loader.Load(c.Context, cfg.Path, e.stdin)
	if err != nil {
		return err
	}
	pl, err := e.deps.Parser.Parse(raw)
	if err != nil {
		return err
	}
	return writeRendered(e, cfg, func(w io.Writer) error { return usage.GenerateSkeleton(pl, w) })
}

func (e *env) liveVerifyAction(c *cli.Context) error {
	cfg, _, warns, err := resolveConfig(c, e.noColor())
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel, e.stderr)
	logWarnings(log, warns)

	if !cfg.Live {
		return fmt.Errorf("live verification requires --live or PRICE_CHECKER_LIVE=1")
	}

	sdkCfg, err := awsauth.Load(c.Context, cfg.Profile, cfg.AWSRegion)
	if err != nil {
		return fmt.Errorf("load AWS config for live verification: %w", err)
	}
	endpointRegion := awsauth.PriceListRegion(sdkCfg)
	targetRegion := awsauth.TargetRegion(sdkCfg, cfg.AWSRegion)

	sdkCfg.Region = endpointRegion
	api := awspricing.NewFromConfig(sdkCfg)

	out, err := api.GetProducts(c.Context, &awspricing.GetProductsInput{
		ServiceCode:   aws.String("AmazonEC2"),
		FormatVersion: aws.String("aws_v1"),
		Filters: []pricingtypes.Filter{
			{Type: pricingtypes.FilterTypeTermMatch, Field: aws.String("regionCode"), Value: aws.String(targetRegion)},
			{Type: pricingtypes.FilterTypeTermMatch, Field: aws.String("instanceType"), Value: aws.String("m5.large")},
		},
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return fmt.Errorf("live AWS Price List verification failed: %w", err)
	}
	if len(out.PriceList) == 0 {
		return fmt.Errorf("live AWS Price List verification returned no products for AmazonEC2/m5.large in %s", targetRegion)
	}

	fmt.Fprintf(e.stdout, "live AWS verification OK: %d Price List products returned for AmazonEC2/m5.large in %s (API endpoint: %s)\n", len(out.PriceList), targetRegion, endpointRegion)
	return nil
}

// ---- helpers ----

func (e *env) request(cfg config.Config) estimator.EstimateRequest {
	return estimator.EstimateRequest{
		Source:         cfg.Path,
		Stdin:          e.stdin,
		RegionOverride: cfg.AWSRegion,
		UsagePath:      cfg.UsageFile,
		Concurrency:    cfg.Concurrency,
		Period:         renderOptions(cfg).Period,
	}
}

func (e *env) estimateOrLoad(ctx context.Context, est Estimator, cfg config.Config, source string) (*schema.Breakdown, error) {
	kind, _ := Classify(source)
	if kind == InputPriorJSON {
		b, err := os.ReadFile(source)
		if err != nil {
			return nil, err
		}
		var bd schema.Breakdown
		if err := json.Unmarshal(b, &bd); err != nil {
			return nil, fmt.Errorf("read prior price-checker JSON %s: %w", source, err)
		}
		bd.Canonicalize()
		return &bd, nil
	}
	req := e.request(cfg)
	req.Source = source
	req.Stdin = e.stdin
	return est.Estimate(ctx, req)
}

func writeRendered(e *env, cfg config.Config, render func(io.Writer) error) error {
	var buf bytes.Buffer
	if err := render(&buf); err != nil {
		return err
	}
	if cfg.Out == "" {
		_, err := io.Copy(e.stdout, &buf)
		return err
	}
	if dir := filepath.Dir(cfg.Out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(cfg.Out, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(e.stderr, "wrote %s\n", cfg.Out)
	return nil
}

// exitFor prints nothing more; it converts a non-zero exit code into a
// cli.ExitCoder so run.go returns it, or nil for success.
func exitFor(o Outcome, cfg config.Config) error {
	code := Evaluate(o, cfg)
	if code == ExitOK {
		return nil
	}
	return cli.Exit("", code)
}

func (e *env) noColor() bool { return e.forceNoColor }

func logWarnings(log *slog.Logger, warns []string) {
	for _, w := range warns {
		log.Warn(w)
	}
}

func logProvenance(log *slog.Logger, cfg config.Config, prov config.Provenance) {
	if !log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	log.Debug("resolved configuration",
		"format", string2(cfg.Format)+" ("+string(prov["format"])+")",
		"period", string2(cfg.Period)+" ("+string(prov["period"])+")",
		"cache_ttl", cfg.CacheTTL.String()+" ("+string(prov["cache-ttl"])+")",
		"concurrency", strconv.Itoa(cfg.Concurrency)+" ("+string(prov["concurrency"])+")",
		"strict", strconv.FormatBool(cfg.Strict)+" ("+string(prov["strict"])+")",
		"log_level", string2(cfg.LogLevel)+" ("+string(prov["log-level"])+")",
	)
}

func string2(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}
