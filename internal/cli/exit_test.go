package cli

import (
	"errors"
	"testing"

	"github.com/osifotos/price-checker/internal/config"
)

func TestEvaluate(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name string
		o    Outcome
		cfg  config.Config
		want int
	}{
		{"success", Outcome{}, config.Config{}, ExitOK},
		{"runtime error", Outcome{Err: errors.New("x")}, config.Config{}, ExitRuntimeErr},
		{"strict trips", Outcome{NotEstimatedCount: 1}, config.Config{Strict: true}, ExitStrict},
		{"strict but nothing missing", Outcome{NotEstimatedCount: 0}, config.Config{Strict: true}, ExitOK},
		{"over monthly threshold", Outcome{MonthlyTotal: 100}, config.Config{ThresholdMonthly: 50}, ExitThreshold},
		{"under monthly threshold", Outcome{MonthlyTotal: 40}, config.Config{ThresholdMonthly: 50}, ExitOK},
		{"over diff threshold", Outcome{MonthlyDelta: f(30)}, config.Config{ThresholdDiffMonthly: 10}, ExitThreshold},
		{"error beats strict", Outcome{Err: errors.New("x"), NotEstimatedCount: 1}, config.Config{Strict: true}, ExitRuntimeErr},
		{"strict beats threshold", Outcome{NotEstimatedCount: 1, MonthlyTotal: 100}, config.Config{Strict: true, ThresholdMonthly: 50}, ExitStrict},
	}
	for _, c := range cases {
		if got := Evaluate(c.o, c.cfg); got != c.want {
			t.Errorf("%s: Evaluate = %d, want %d", c.name, got, c.want)
		}
	}
}
