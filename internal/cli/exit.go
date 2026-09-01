package cli

import "github.com/mtosin123/tf-price_checker/internal/config"

// Exit codes. Stable across patch releases and documented in --help / README.
const (
	ExitOK         = 0 // success
	ExitRuntimeErr = 1 // bad input, unreadable plan, credentials, etc.
	ExitStrict     = 2 // --strict and at least one component not estimated
	ExitThreshold  = 3 // a cost threshold was breached
)

// Outcome summarises a completed estimation for the exit-code decision.
type Outcome struct {
	Err               error
	NotEstimatedCount int
	MonthlyTotal      float64
	MonthlyDelta      *float64 // nil unless this was a diff
}

// Evaluate maps an outcome and config to a process exit code. Precedence:
// runtime error (1) > strict (2) > threshold (3).
func Evaluate(o Outcome, cfg config.Config) int {
	if o.Err != nil {
		return ExitRuntimeErr
	}
	if cfg.Strict && o.NotEstimatedCount > 0 {
		return ExitStrict
	}
	if cfg.ThresholdMonthly > 0 && o.MonthlyTotal > cfg.ThresholdMonthly {
		return ExitThreshold
	}
	if cfg.ThresholdDiffMonthly > 0 && o.MonthlyDelta != nil && *o.MonthlyDelta > cfg.ThresholdDiffMonthly {
		return ExitThreshold
	}
	return ExitOK
}
