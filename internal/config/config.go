// Package config resolves price-checker settings from, in order of precedence,
// command-line flags / environment variables, an optional YAML config file, and
// built-in defaults. It records the provenance of every value.
package config

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the fully resolved settings object.
type Config struct {
	Path                 string
	CompareTo            string
	FromPlan             bool
	Format               string
	Out                  string
	Period               string
	UsageFile            string
	AWSRegion            string
	Profile              string
	Live                 bool
	CacheDir             string
	CacheTTL             time.Duration
	NoCache              bool
	RefreshCache         bool
	Strict               bool
	ThresholdMonthly     float64
	ThresholdDiffMonthly float64
	ShowComponents       bool
	Concurrency          int
	LogLevel             string
	NoColor              bool
}

// Source is where a setting's value came from.
type Source string

const (
	SourceFlag    Source = "flag/env"
	SourceFile    Source = "config file"
	SourceDefault Source = "default"
)

// Provenance maps a setting name to its Source.
type Provenance map[string]Source

// Raw is one setting as seen on the command line.
type Raw struct {
	Value any
	Set   bool // true when set by a flag or an environment variable
}

// Inputs is the flag/env layer plus the config-file path.
type Inputs struct {
	Values     map[string]Raw
	ConfigPath string // explicit --config; "" means try ./price-checker.yml
}

var defaults = Config{
	Format:      "table",
	Period:      "month",
	CacheTTL:    7 * 24 * time.Hour,
	Concurrency: 8,
	LogLevel:    "warn",
}

type fileConfig struct {
	Path                 *string  `yaml:"path"`
	CompareTo            *string  `yaml:"compare_to"`
	FromPlan             *bool    `yaml:"from_plan"`
	Format               *string  `yaml:"format"`
	Out                  *string  `yaml:"out"`
	Period               *string  `yaml:"period"`
	UsageFile            *string  `yaml:"usage_file"`
	AWSRegion            *string  `yaml:"aws_region"`
	Profile              *string  `yaml:"profile"`
	Live                 *bool    `yaml:"live"`
	CacheDir             *string  `yaml:"cache_dir"`
	CacheTTL             *string  `yaml:"cache_ttl"`
	NoCache              *bool    `yaml:"no_cache"`
	RefreshCache         *bool    `yaml:"refresh_cache"`
	Strict               *bool    `yaml:"strict"`
	ThresholdMonthly     *float64 `yaml:"threshold_monthly"`
	ThresholdDiffMonthly *float64 `yaml:"threshold_diff_monthly"`
	ShowComponents       *bool    `yaml:"show_components"`
	Concurrency          *int     `yaml:"concurrency"`
	LogLevel             *string  `yaml:"log_level"`
	NoColor              *bool    `yaml:"no_color"`
}

// Resolve merges the layers and returns the config, its provenance, and any
// non-fatal warnings (for example unknown config-file keys).
func Resolve(in Inputs) (Config, Provenance, []string, error) {
	cfg := defaults
	prov := Provenance{}
	for k := range fieldNames {
		prov[k] = SourceDefault
	}

	var warnings []string
	fc, fcWarn, err := loadFile(in.ConfigPath)
	if err != nil {
		return Config{}, nil, nil, err
	}
	warnings = append(warnings, fcWarn...)

	applyFile(&cfg, prov, fc)
	if aw := applyFlags(&cfg, prov, in.Values); len(aw) > 0 {
		warnings = append(warnings, aw...)
	}
	return cfg, prov, warnings, nil
}

func loadFile(explicit string) (fileConfig, []string, error) {
	path := explicit
	if path == "" {
		if _, err := os.Stat("price-checker.yml"); err == nil {
			path = "price-checker.yml"
		} else {
			return fileConfig{}, nil, nil
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	// strict pass to detect unknown keys as warnings
	var warnings []string
	strict := yaml.NewDecoder(bytes.NewReader(b))
	strict.KnownFields(true)
	var probe fileConfig
	if err := strict.Decode(&probe); err != nil {
		warnings = append(warnings, fmt.Sprintf("config file %s: %v", path, err))
	}

	var fc fileConfig
	if err := yaml.Unmarshal(b, &fc); err != nil {
		return fileConfig{}, warnings, fmt.Errorf("parse config file %s: %w", path, err)
	}
	return fc, warnings, nil
}

func applyFile(cfg *Config, prov Provenance, fc fileConfig) {
	set := func(name string, apply func()) {
		apply()
		prov[name] = SourceFile
	}
	if fc.Path != nil {
		set("path", func() { cfg.Path = *fc.Path })
	}
	if fc.CompareTo != nil {
		set("compare-to", func() { cfg.CompareTo = *fc.CompareTo })
	}
	if fc.FromPlan != nil {
		set("from-plan", func() { cfg.FromPlan = *fc.FromPlan })
	}
	if fc.Format != nil {
		set("format", func() { cfg.Format = *fc.Format })
	}
	if fc.Out != nil {
		set("out", func() { cfg.Out = *fc.Out })
	}
	if fc.Period != nil {
		set("period", func() { cfg.Period = *fc.Period })
	}
	if fc.UsageFile != nil {
		set("usage-file", func() { cfg.UsageFile = *fc.UsageFile })
	}
	if fc.AWSRegion != nil {
		set("aws-region", func() { cfg.AWSRegion = *fc.AWSRegion })
	}
	if fc.Profile != nil {
		set("profile", func() { cfg.Profile = *fc.Profile })
	}
	if fc.Live != nil {
		set("live", func() { cfg.Live = *fc.Live })
	}
	if fc.CacheDir != nil {
		set("cache-dir", func() { cfg.CacheDir = *fc.CacheDir })
	}
	if fc.CacheTTL != nil {
		if d, err := time.ParseDuration(*fc.CacheTTL); err == nil {
			set("cache-ttl", func() { cfg.CacheTTL = d })
		}
	}
	if fc.NoCache != nil {
		set("no-cache", func() { cfg.NoCache = *fc.NoCache })
	}
	if fc.RefreshCache != nil {
		set("refresh-cache", func() { cfg.RefreshCache = *fc.RefreshCache })
	}
	if fc.Strict != nil {
		set("strict", func() { cfg.Strict = *fc.Strict })
	}
	if fc.ThresholdMonthly != nil {
		set("threshold-monthly", func() { cfg.ThresholdMonthly = *fc.ThresholdMonthly })
	}
	if fc.ThresholdDiffMonthly != nil {
		set("threshold-diff-monthly", func() { cfg.ThresholdDiffMonthly = *fc.ThresholdDiffMonthly })
	}
	if fc.ShowComponents != nil {
		set("show-components", func() { cfg.ShowComponents = *fc.ShowComponents })
	}
	if fc.Concurrency != nil {
		set("concurrency", func() { cfg.Concurrency = *fc.Concurrency })
	}
	if fc.LogLevel != nil {
		set("log-level", func() { cfg.LogLevel = *fc.LogLevel })
	}
	if fc.NoColor != nil {
		set("no-color", func() { cfg.NoColor = *fc.NoColor })
	}
}

func applyFlags(cfg *Config, prov Provenance, vals map[string]Raw) []string {
	var warnings []string
	get := func(name string) (Raw, bool) {
		r, ok := vals[name]
		return r, ok && r.Set
	}
	setS := func(name string, dst *string) {
		if r, ok := get(name); ok {
			*dst, _ = r.Value.(string)
			prov[name] = SourceFlag
		}
	}
	setB := func(name string, dst *bool) {
		if r, ok := get(name); ok {
			*dst, _ = r.Value.(bool)
			prov[name] = SourceFlag
		}
	}
	setF := func(name string, dst *float64) {
		if r, ok := get(name); ok {
			*dst, _ = r.Value.(float64)
			prov[name] = SourceFlag
		}
	}
	setI := func(name string, dst *int) {
		if r, ok := get(name); ok {
			*dst, _ = r.Value.(int)
			prov[name] = SourceFlag
		}
	}
	setD := func(name string, dst *time.Duration) {
		if r, ok := get(name); ok {
			if d, okd := r.Value.(time.Duration); okd {
				*dst = d
				prov[name] = SourceFlag
			}
		}
	}

	setS("path", &cfg.Path)
	setS("compare-to", &cfg.CompareTo)
	setB("from-plan", &cfg.FromPlan)
	setS("format", &cfg.Format)
	setS("out", &cfg.Out)
	setS("period", &cfg.Period)
	setS("usage-file", &cfg.UsageFile)
	setS("aws-region", &cfg.AWSRegion)
	setS("profile", &cfg.Profile)
	setB("live", &cfg.Live)
	setS("cache-dir", &cfg.CacheDir)
	setD("cache-ttl", &cfg.CacheTTL)
	setB("no-cache", &cfg.NoCache)
	setB("refresh-cache", &cfg.RefreshCache)
	setB("strict", &cfg.Strict)
	setF("threshold-monthly", &cfg.ThresholdMonthly)
	setF("threshold-diff-monthly", &cfg.ThresholdDiffMonthly)
	setB("show-components", &cfg.ShowComponents)
	setI("concurrency", &cfg.Concurrency)
	setS("log-level", &cfg.LogLevel)
	setB("no-color", &cfg.NoColor)
	return warnings
}

var fieldNames = map[string]struct{}{
	"path": {}, "compare-to": {}, "from-plan": {}, "format": {}, "out": {}, "period": {},
	"usage-file": {}, "aws-region": {}, "profile": {}, "live": {}, "cache-dir": {}, "cache-ttl": {},
	"no-cache": {}, "refresh-cache": {}, "strict": {}, "threshold-monthly": {},
	"threshold-diff-monthly": {}, "show-components": {}, "concurrency": {}, "log-level": {}, "no-color": {},
}
