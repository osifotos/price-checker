package cli

import (
	"fmt"
	"runtime"
)

// Build metadata, overridden at build time with
// -ldflags "-X github.com/mtosin123/tf-price_checker/internal/cli.version=..."
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Version returns the short version string.
func Version() string { return version }

func versionString() string {
	return fmt.Sprintf("price-checker %s (commit %s, built %s, %s)", version, commit, date, runtime.Version())
}
