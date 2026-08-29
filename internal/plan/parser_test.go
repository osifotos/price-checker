package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func loadFixture(t *testing.T, name string) *Plan {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	p, err := NewParser().Parse(raw)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return p
}

func TestParseFixture(t *testing.T) {
	p := loadFixture(t, "ec2_s3_natgw.json")

	if p.FormatVersion != "1.2" || p.TerraformVersion != "1.9.5" {
		t.Fatalf("version fields wrong: %+v", p)
	}

	got := map[string]Resource{}
	for _, r := range p.Resources() {
		got[r.Address] = r
	}
	// pure delete (aws_eip.old) and data source excluded; count expanded; replace kept
	want := []string{
		"aws_db_instance.main",
		"aws_instance.web[0]",
		"aws_instance.web[1]",
		"aws_s3_bucket.assets",
		"module.net.aws_nat_gateway.this",
	}
	if len(got) != len(want) {
		t.Fatalf("Resources() = %d entries, want %d (%v)", len(got), len(want), keysOf(got))
	}
	for _, w := range want {
		if _, ok := got[w]; !ok {
			t.Errorf("missing resource %q", w)
		}
	}

	// determinism
	first := p.Resources()
	second := p.Resources()
	for i := range first {
		if first[i].Address != second[i].Address {
			t.Fatalf("Resources() not stably ordered")
		}
	}
	if first[0].Address != "aws_db_instance.main" {
		t.Fatalf("Resources() not sorted: %s first", first[0].Address)
	}
}

func TestParseActions(t *testing.T) {
	p := loadFixture(t, "ec2_s3_natgw.json")
	byAddr := map[string]ChangeAction{}
	for _, r := range p.AllResources() {
		byAddr[r.Address] = r.Action
	}
	cases := map[string]ChangeAction{
		"aws_instance.web[0]":             ActionCreate,
		"aws_eip.old":                     ActionDelete,
		"data.aws_ami.ubuntu":             ActionRead,
		"aws_db_instance.main":            ActionReplace,
		"module.net.aws_nat_gateway.this": ActionCreate,
	}
	for addr, want := range cases {
		if byAddr[addr] != want {
			t.Errorf("%s action = %q, want %q", addr, byAddr[addr], want)
		}
	}
}

func TestParseModuleAddressAndProvider(t *testing.T) {
	p := loadFixture(t, "ec2_s3_natgw.json")
	for _, r := range p.AllResources() {
		switch r.Address {
		case "module.net.aws_nat_gateway.this":
			if r.ModuleAddress != "module.net" {
				t.Errorf("nat gateway ModuleAddress = %q", r.ModuleAddress)
			}
			if r.ProviderConfigKey != "aws" {
				t.Errorf("nat gateway ProviderConfigKey = %q, want aws", r.ProviderConfigKey)
			}
		case "aws_db_instance.main":
			if r.ProviderConfigKey != "aws.west" {
				t.Errorf("db ProviderConfigKey = %q, want aws.west", r.ProviderConfigKey)
			}
		case "aws_instance.web[0]":
			if r.ProviderConfigKey != "aws" {
				t.Errorf("indexed instance ProviderConfigKey = %q, want aws (index stripped)", r.ProviderConfigKey)
			}
		}
	}
}

func TestProviderConfigRegion(t *testing.T) {
	p := loadFixture(t, "ec2_s3_natgw.json")
	if p.ProviderConfigs["aws"].ConstantRegion != "us-east-1" {
		t.Errorf("aws region = %q", p.ProviderConfigs["aws"].ConstantRegion)
	}
	if p.ProviderConfigs["aws.west"].ConstantRegion != "us-west-2" {
		t.Errorf("aws.west region = %q", p.ProviderConfigs["aws.west"].ConstantRegion)
	}
	if p.ProviderConfigs["aws.west"].Alias != "west" {
		t.Errorf("aws.west alias = %q", p.ProviderConfigs["aws.west"].Alias)
	}
}

func TestPriorResources(t *testing.T) {
	p := loadFixture(t, "ec2_s3_natgw.json")
	prior := map[string]bool{}
	for _, r := range p.PriorResources() {
		prior[r.Address] = true
	}
	// db is a replace (has before); eip is a delete (has before)
	if !prior["aws_db_instance.main"] {
		t.Error("expected aws_db_instance.main in PriorResources")
	}
	// creates have no prior
	if prior["aws_instance.web[0]"] {
		t.Error("create should not be in PriorResources")
	}
}

func TestFormatVersionValidation(t *testing.T) {
	cases := []struct {
		body    string
		wantErr bool
	}{
		{`{"format_version":"1.2","resource_changes":[]}`, false},
		{`{"format_version":"0.1","resource_changes":[]}`, false},
		{`{"format_version":"0.2"}`, false},
		{`{"format_version":"2.0"}`, true},
		{`{"format_version":"99.0"}`, true},
		{`{"format_version":""}`, true},
		{`{"resource_changes":[]}`, true},
		{`{`, true},
		{`not json`, true},
	}
	for _, c := range cases {
		_, err := NewParser().Parse([]byte(c.body))
		if (err != nil) != c.wantErr {
			t.Errorf("Parse(%q) err=%v, wantErr=%v", c.body, err, c.wantErr)
		}
	}
}

func TestEmptyPlanIsValid(t *testing.T) {
	p, err := NewParser().Parse([]byte(`{"format_version":"1.2","terraform_version":"1.9.0"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Resources()) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(p.Resources()))
	}
}

func keysOf(m map[string]Resource) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
