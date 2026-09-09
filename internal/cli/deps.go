package cli

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/osifotos/price-checker/internal/config"
	"github.com/osifotos/price-checker/internal/estimator"
	"github.com/osifotos/price-checker/internal/plan"
	"github.com/osifotos/price-checker/pkg/schema"
)

// Estimator is the slice of *estimator.Estimator the CLI needs; a fake is
// injected in tests.
type Estimator interface {
	Estimate(ctx context.Context, req estimator.EstimateRequest) (*schema.Breakdown, error)
	EstimateBothStates(ctx context.Context, req estimator.EstimateRequest) (*schema.Breakdown, *schema.Breakdown, error)
}

// Deps are the CLI's swappable collaborators. A nil *Deps means production.
type Deps struct {
	// NewEstimator builds a live estimator (AWS-backed). Returns a cleanup func.
	NewEstimator func(ctx context.Context, cfg config.Config, log *slog.Logger) (Estimator, func(), error)
	// Loader / Parser are used by `usage generate`, which does no pricing.
	Loader plan.PlanLoader
	Parser plan.PlanParser
	Clock  func() time.Time
}

func (d *Deps) withDefaults() *Deps {
	out := Deps{}
	if d != nil {
		out = *d
	}
	if out.NewEstimator == nil {
		out.NewEstimator = buildEstimator
	}
	if out.Loader == nil {
		out.Loader = plan.NewLoader()
	}
	if out.Parser == nil {
		out.Parser = plan.NewParser()
	}
	if out.Clock == nil {
		out.Clock = time.Now
	}
	return &out
}

func newLogger(level string, stderr io.Writer) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelWarn
	}
	return slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: l}))
}
