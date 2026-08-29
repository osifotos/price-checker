package cli

import (
	"io"

	"github.com/urfave/cli/v2"

	"github.com/example/price-checker/internal/config"
)

type env struct {
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	deps         *Deps
	forceNoColor bool
}

func buildApp(e *env) *cli.App {
	app := &cli.App{
		Name:        "price-checker",
		Usage:       "estimate the AWS cost of a Terraform plan",
		Version:     versionString(),
		HideVersion: true, // we expose `price-checker version`; frees up -v for --show-components
		Writer:      e.stdout,
		ErrWriter:   e.stderr,
		Flags:       globalFlags(),
		Action:      e.breakdownAction, // default command
		// Do not let urfave call os.Exit; runWithDeps maps the ExitCoder itself.
		ExitErrHandler: func(*cli.Context, error) {},
		Commands: []*cli.Command{
			{Name: "breakdown", Usage: "cost breakdown of a plan (default)", Action: e.breakdownAction, Flags: globalFlags()},
			{Name: "diff", Usage: "cost change between two plans / prior outputs", Action: e.diffAction, Flags: globalFlags()},
			{Name: "report", Usage: "write a self-contained HTML report", Action: e.reportAction, Flags: globalFlags()},
			{
				Name:  "usage",
				Usage: "usage-file helpers",
				Subcommands: []*cli.Command{
					{Name: "generate", Usage: "print a skeleton usage file for a plan", Action: e.usageGenerateAction, Flags: globalFlags()},
				},
			},
			{Name: "version", Usage: "print version information", Action: e.versionAction},
		},
		CustomAppHelpTemplate: helpTemplate,
	}
	return app
}

func globalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "path", Aliases: []string{"p"}, EnvVars: []string{"PRICE_CHECKER_PATH"}, Usage: "plan JSON file, Terraform directory, or '-' for stdin"},
		&cli.StringFlag{Name: "compare-to", EnvVars: []string{"PRICE_CHECKER_COMPARE_TO"}, Usage: "baseline plan / prior price-checker JSON (diff)"},
		&cli.BoolFlag{Name: "from-plan", EnvVars: []string{"PRICE_CHECKER_FROM_PLAN"}, Usage: "derive the diff from prior+planned state in one plan"},
		&cli.StringFlag{Name: "format", Aliases: []string{"f"}, EnvVars: []string{"PRICE_CHECKER_FORMAT"}, Usage: "table | json | html | github-comment"},
		&cli.StringFlag{Name: "out", Aliases: []string{"o"}, EnvVars: []string{"PRICE_CHECKER_OUT"}, Usage: "write output to a file instead of stdout"},
		&cli.StringFlag{Name: "period", EnvVars: []string{"PRICE_CHECKER_PERIOD"}, Usage: "month | hour"},
		&cli.StringFlag{Name: "usage-file", EnvVars: []string{"PRICE_CHECKER_USAGE_FILE"}, Usage: "usage assumptions YAML"},
		&cli.StringFlag{Name: "aws-region", EnvVars: []string{"PRICE_CHECKER_AWS_REGION"}, Usage: "override the region for every resource"},
		&cli.StringFlag{Name: "profile", EnvVars: []string{"AWS_PROFILE"}, Usage: "AWS shared-config profile"},
		&cli.StringFlag{Name: "config", EnvVars: []string{"PRICE_CHECKER_CONFIG"}, Usage: "config file (default ./price-checker.yml)"},
		&cli.StringFlag{Name: "cache-dir", EnvVars: []string{"PRICE_CHECKER_CACHE_DIR"}, Usage: "price cache directory"},
		&cli.DurationFlag{Name: "cache-ttl", EnvVars: []string{"PRICE_CHECKER_CACHE_TTL"}, Usage: "max age of a cached price (default 168h)"},
		&cli.BoolFlag{Name: "no-cache", EnvVars: []string{"PRICE_CHECKER_NO_CACHE"}, Usage: "bypass the price cache"},
		&cli.BoolFlag{Name: "refresh-cache", EnvVars: []string{"PRICE_CHECKER_REFRESH_CACHE"}, Usage: "ignore cached prices but refresh them"},
		&cli.BoolFlag{Name: "strict", EnvVars: []string{"PRICE_CHECKER_STRICT"}, Usage: "exit 2 if anything could not be estimated"},
		&cli.Float64Flag{Name: "threshold-monthly", EnvVars: []string{"PRICE_CHECKER_THRESHOLD_MONTHLY"}, Usage: "exit 3 if the monthly total exceeds this"},
		&cli.Float64Flag{Name: "threshold-diff-monthly", EnvVars: []string{"PRICE_CHECKER_THRESHOLD_DIFF_MONTHLY"}, Usage: "exit 3 if the monthly delta exceeds this"},
		&cli.BoolFlag{Name: "show-components", Aliases: []string{"v"}, EnvVars: []string{"PRICE_CHECKER_SHOW_COMPONENTS"}, Usage: "show per-resource cost components"},
		&cli.IntFlag{Name: "concurrency", EnvVars: []string{"PRICE_CHECKER_CONCURRENCY"}, Usage: "parallel pricing requests (default 8)"},
		&cli.StringFlag{Name: "log-level", EnvVars: []string{"PRICE_CHECKER_LOG_LEVEL"}, Usage: "error | warn | info | debug"},
		&cli.BoolFlag{Name: "no-color", EnvVars: []string{"NO_COLOR"}, Usage: "disable ANSI colour"},
	}
}

// resolveConfig turns a cli.Context into a resolved Config.
func resolveConfig(c *cli.Context, noColorForced bool) (config.Config, config.Provenance, []string, error) {
	vals := map[string]config.Raw{
		"path":                   {Value: c.String("path"), Set: c.IsSet("path")},
		"compare-to":             {Value: c.String("compare-to"), Set: c.IsSet("compare-to")},
		"from-plan":              {Value: c.Bool("from-plan"), Set: c.IsSet("from-plan")},
		"format":                 {Value: c.String("format"), Set: c.IsSet("format")},
		"out":                    {Value: c.String("out"), Set: c.IsSet("out")},
		"period":                 {Value: c.String("period"), Set: c.IsSet("period")},
		"usage-file":             {Value: c.String("usage-file"), Set: c.IsSet("usage-file")},
		"aws-region":             {Value: c.String("aws-region"), Set: c.IsSet("aws-region")},
		"profile":                {Value: c.String("profile"), Set: c.IsSet("profile")},
		"cache-dir":              {Value: c.String("cache-dir"), Set: c.IsSet("cache-dir")},
		"cache-ttl":              {Value: c.Duration("cache-ttl"), Set: c.IsSet("cache-ttl")},
		"no-cache":               {Value: c.Bool("no-cache"), Set: c.IsSet("no-cache")},
		"refresh-cache":          {Value: c.Bool("refresh-cache"), Set: c.IsSet("refresh-cache")},
		"strict":                 {Value: c.Bool("strict"), Set: c.IsSet("strict")},
		"threshold-monthly":      {Value: c.Float64("threshold-monthly"), Set: c.IsSet("threshold-monthly")},
		"threshold-diff-monthly": {Value: c.Float64("threshold-diff-monthly"), Set: c.IsSet("threshold-diff-monthly")},
		"show-components":        {Value: c.Bool("show-components"), Set: c.IsSet("show-components")},
		"concurrency":            {Value: c.Int("concurrency"), Set: c.IsSet("concurrency")},
		"log-level":              {Value: c.String("log-level"), Set: c.IsSet("log-level")},
		"no-color":               {Value: c.Bool("no-color"), Set: c.IsSet("no-color")},
	}
	cfg, prov, warns, err := config.Resolve(config.Inputs{Values: vals, ConfigPath: c.String("config")})
	if err != nil {
		return config.Config{}, nil, nil, err
	}
	if noColorForced {
		cfg.NoColor = true
		prov["no-color"] = config.SourceFlag
	}
	return cfg, prov, warns, nil
}

const helpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   price-checker [global options] [command]

COMMANDS:
   breakdown       cost breakdown of a plan (default)
   diff            cost change between two plans / prior outputs
   report          write a self-contained HTML report
   usage generate  print a skeleton usage file
   version         print version information

EXIT CODES:
   0  success
   1  runtime error (bad input, unreadable plan, credentials)
   2  --strict and at least one component was not estimated
   3  a --threshold-* limit was exceeded

GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
`
