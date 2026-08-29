// Package cli is the price-checker command-line program: flag/config
// resolution, the urfave/cli command tree, wiring of the U0-U4 packages, and
// the exit-code policy.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v2"
)

// Run builds and runs the CLI and returns the process exit code. main() is a
// one-liner over this.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runWithDeps(args, stdin, stdout, stderr, nil)
}

func runWithDeps(args []string, stdin io.Reader, stdout, stderr io.Writer, deps *Deps) int {
	e := &env{
		stdin:        stdin,
		stdout:       stdout,
		stderr:       stderr,
		deps:         deps.withDefaults(),
		forceNoColor: noColorForced(stdout),
	}

	app := buildApp(e)
	err := app.Run(args)
	if err == nil {
		return ExitOK
	}

	var coder cli.ExitCoder
	if errors.As(err, &coder) {
		if msg := coder.Error(); msg != "" {
			fmt.Fprintln(stderr, msg)
		}
		return coder.ExitCode()
	}
	fmt.Fprintln(stderr, "error:", err)
	return ExitRuntimeErr
}

// noColorForced reports whether colour should be disabled regardless of the
// --no-color flag: the NO_COLOR env var is set, or stdout is not a terminal.
func noColorForced(stdout io.Writer) bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return true
	}
	f, ok := stdout.(*os.File)
	if !ok {
		return true // a buffer / pipe in tests
	}
	info, err := f.Stat()
	if err != nil {
		return true
	}
	return info.Mode()&os.ModeCharDevice == 0
}
