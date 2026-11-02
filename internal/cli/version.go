package cli

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Build metadata, overridden by GoReleaser / the release workflow with
// -ldflags "-X github.com/osifotos/price-checker/internal/cli.version=..."
var (
	version = "v0.1.5"
	commit  = "none"
	date    = "unknown"
)

// Version returns the resolved version string, preferring ldflags, then the
// module version recorded by `go install`, then "dev".
func Version() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}

func versionString() string {
	v := Version()
	c, d := commit, date
	if c == "none" || d == "unknown" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					if c == "none" && s.Value != "" {
						c = s.Value
					}
				case "vcs.time":
					if d == "unknown" && s.Value != "" {
						d = s.Value
					}
				}
			}
		}
	}
	return fmt.Sprintf("price-checker %s (commit %s, built %s, %s)", v, c, d, runtime.Version())
}
