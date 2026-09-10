package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolvePrecedence(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "price-checker.yml")
	if err := os.WriteFile(cfgPath, []byte("format: json\nconcurrency: 4\nperiod: hour\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	in := Inputs{
		ConfigPath: cfgPath,
		Values: map[string]Raw{
			"format":      {Value: "table", Set: true}, // flag beats file
			"concurrency": {Value: 0, Set: false},      // not set -> file wins
		},
	}
	cfg, prov, warns, err := Resolve(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	if cfg.Format != "table" || prov["format"] != SourceFlag {
		t.Errorf("format = %q (%s), want table (flag/env)", cfg.Format, prov["format"])
	}
	if cfg.Concurrency != 4 || prov["concurrency"] != SourceFile {
		t.Errorf("concurrency = %d (%s), want 4 (config file)", cfg.Concurrency, prov["concurrency"])
	}
	if cfg.Period != "hour" || prov["period"] != SourceFile {
		t.Errorf("period = %q (%s)", cfg.Period, prov["period"])
	}
	if cfg.CacheTTL != 7*24*time.Hour || prov["cache-ttl"] != SourceDefault {
		t.Errorf("cache-ttl = %s (%s), want default 168h", cfg.CacheTTL, prov["cache-ttl"])
	}
}

func TestResolveNoFile(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(old); err != nil {
			t.Fatal(err)
		}
	}()

	cfg, prov, warns, err := Resolve(Inputs{Values: map[string]Raw{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Errorf("warnings: %v", warns)
	}
	if cfg.Format != "table" || prov["format"] != SourceDefault {
		t.Errorf("default format wrong: %q %s", cfg.Format, prov["format"])
	}
}

func TestResolveLiveMode(t *testing.T) {
	cfg, _, warns, err := Resolve(Inputs{Values: map[string]Raw{
		"live": {Value: true, Set: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	if !cfg.Live {
		t.Fatal("expected live mode to be enabled")
	}
	if cfg.Live && cfg.Format != "table" {
		t.Fatalf("live mode should not change the default format: got %q", cfg.Format)
	}
}

func TestResolveUnknownKeyWarns(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yml")
	if err := os.WriteFile(p, []byte("format: json\nbogus: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, warns, err := Resolve(Inputs{ConfigPath: p, Values: map[string]Raw{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) == 0 {
		t.Error("expected a warning for the unknown key")
	}
}

func TestResolveBadYAML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yml")
	if err := os.WriteFile(p, []byte("format: [unterminated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := Resolve(Inputs{ConfigPath: p, Values: map[string]Raw{}}); err == nil {
		t.Fatal("expected error for bad YAML")
	}
}
