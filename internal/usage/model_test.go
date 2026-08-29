package usage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTmp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "usage.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadEmptyPath(t *testing.T) {
	m, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if m.Used() {
		t.Error("empty path model should not be Used()")
	}
	if len(m.For("anything")) != 0 {
		t.Error("For on empty model should be empty")
	}
}

func TestLoadValid(t *testing.T) {
	p := writeTmp(t, `version: "0.1"
resource_usage:
  aws_s3_bucket.assets:
    storage_gb: 512
    monthly_tier1_requests: 100000
`)
	m, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Used() {
		t.Error("expected Used()")
	}
	ru := m.For("aws_s3_bucket.assets")
	if ru["storage_gb"] != 512 {
		t.Errorf("storage_gb = %v", ru["storage_gb"])
	}
	if len(m.Warnings()) != 0 {
		t.Errorf("unexpected warnings: %v", m.Warnings())
	}
}

func TestLoadUnknownKeyWarns(t *testing.T) {
	p := writeTmp(t, `version: "0.1"
resource_usage:
  aws_s3_bucket.assets:
    storage_gb: 10
    bogus_key: 5
`)
	m, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Warnings()) != 1 || !strings.Contains(m.Warnings()[0], "bogus_key") {
		t.Errorf("want unknown-key warning, got %v", m.Warnings())
	}
}

func TestValidateUnknownAddressWarns(t *testing.T) {
	p := writeTmp(t, `version: "0.1"
resource_usage:
  aws_s3_bucket.gone:
    storage_gb: 10
`)
	m, _ := Load(p)
	m.Validate(map[string]bool{"aws_s3_bucket.here": true})
	found := false
	for _, w := range m.Warnings() {
		if strings.Contains(w, "aws_s3_bucket.gone") && strings.Contains(w, "not in the plan") {
			found = true
		}
	}
	if !found {
		t.Errorf("want unknown-address warning, got %v", m.Warnings())
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	p := writeTmp(t, "version: \"0.1\"\nresource_usage: [this is not a map")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}
