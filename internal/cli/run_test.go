package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/mtosin123/tf-price_checker/internal/config"
	"github.com/mtosin123/tf-price_checker/internal/estimator"
	"github.com/mtosin123/tf-price_checker/pkg/schema"
	"github.com/mtosin123/tf-price_checker/pkg/schema/schematest"
)

type fakeEst struct {
	b   *schema.Breakdown
	err error
}

func (f fakeEst) Estimate(context.Context, estimator.EstimateRequest) (*schema.Breakdown, error) {
	return f.b, f.err
}
func (f fakeEst) EstimateBothStates(context.Context, estimator.EstimateRequest) (*schema.Breakdown, *schema.Breakdown, error) {
	return f.b, f.b, f.err
}

func depsWith(b *schema.Breakdown) *Deps {
	return &Deps{
		NewEstimator: func(context.Context, config.Config, *slog.Logger) (Estimator, func(), error) {
			return fakeEst{b: b}, func() {}, nil
		},
	}
}

func run(t *testing.T, deps *Deps, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := runWithDeps(append([]string{"price-checker"}, args...), strings.NewReader(""), &out, &errb, deps)
	return out.String(), errb.String(), code
}

func TestRunVersion(t *testing.T) {
	out, _, code := run(t, &Deps{}, "version")
	if code != 0 || !strings.Contains(out, "price-checker") {
		t.Fatalf("version: code=%d out=%q", code, out)
	}
}

func TestRunBreakdownTable(t *testing.T) {
	out, _, code := run(t, depsWith(schematest.SampleBreakdown()), "breakdown")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(out, "PROJECT TOTAL") {
		t.Fatalf("expected table on stdout, got:\n%s", out)
	}
}

func TestRunBreakdownJSON(t *testing.T) {
	out, errb, code := run(t, depsWith(schematest.SampleBreakdown()), "--format", "json")
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, errb)
	}
	var b schema.Breakdown
	if err := json.Unmarshal([]byte(out), &b); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out)
	}
}

func TestRunStrictExit2(t *testing.T) {
	b := schematest.SampleBreakdown() // has 2 not-estimated
	_, _, code := run(t, depsWith(b), "--strict")
	if code != ExitStrict {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestRunThresholdExit3(t *testing.T) {
	b := schematest.SampleBreakdown() // total ~65.62
	out, _, code := run(t, depsWith(b), "--threshold-monthly", "10")
	if code != ExitThreshold {
		t.Fatalf("expected exit 3, got %d", code)
	}
	if !strings.Contains(out, "PROJECT TOTAL") {
		t.Fatalf("full output must be printed before exiting 3:\n%s", out)
	}
}

func TestRunUnknownFormat(t *testing.T) {
	_, errb, code := run(t, depsWith(schematest.SampleBreakdown()), "--format", "xml")
	if code != ExitRuntimeErr {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(errb, "unknown format") {
		t.Fatalf("expected error message on stderr, got %q", errb)
	}
}

func TestRunEstimatorError(t *testing.T) {
	deps := &Deps{
		NewEstimator: func(context.Context, config.Config, *slog.Logger) (Estimator, func(), error) {
			return fakeEst{err: context.DeadlineExceeded}, func() {}, nil
		},
	}
	_, _, code := run(t, deps, "breakdown")
	if code != ExitRuntimeErr {
		t.Fatalf("expected exit 1, got %d", code)
	}
}

func TestRunHelpListsFlagsAndExitCodes(t *testing.T) {
	out, _, _ := run(t, &Deps{}, "--help")
	for _, want := range []string{"--strict", "--threshold-monthly", "--format", "EXIT CODES", "  2  ", "  3  "} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q", want)
		}
	}
}
