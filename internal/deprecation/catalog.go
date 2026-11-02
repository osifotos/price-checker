package deprecation

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/osifotos/price-checker/internal/plan"
	"github.com/osifotos/price-checker/pkg/schema"
	"gopkg.in/yaml.v3"
)

type rule struct {
	Message       string `yaml:"message"`
	Replacement   string `yaml:"replacement"`
	EffectiveDate string `yaml:"effective_date"`
	Reference     string `yaml:"reference"`
}

type rulesFile struct {
	Resources map[string]rule `yaml:"resources"`
}

var builtInRules = map[string]rule{
	"aws_launch_configuration": {
		Message:       "AWS restricts creation of launch configurations for accounts created on or after October 1, 2024. Migrate to launch templates.",
		Replacement:   "aws_launch_template",
		EffectiveDate: "2024-10-01",
		Reference:     "https://docs.aws.amazon.com/autoscaling/ec2/userguide/launch-configurations.html",
	},
}

// Find combines provider diagnostics with AWS lifecycle rules for planned resources.
func Find(resources []plan.Resource, diagnostics []plan.TerraformDeprecation, rulesPath string) ([]schema.Deprecation, error) {
	rules := make(map[string]rule, len(builtInRules))
	for resourceType, entry := range builtInRules {
		rules[resourceType] = entry
	}
	if rulesPath != "" {
		custom, err := loadRules(rulesPath)
		if err != nil {
			return nil, err
		}
		for resourceType, entry := range custom.Resources {
			rules[resourceType] = entry
		}
	}

	var findings []schema.Deprecation
	for _, resource := range resources {
		if entry, ok := rules[resource.Type]; ok {
			findings = append(findings, schema.Deprecation{
				Address: resource.Address, Type: resource.Type, Source: "aws-lifecycle",
				Message: entry.Message, Replacement: entry.Replacement,
				EffectiveDate: entry.EffectiveDate, Reference: entry.Reference,
			})
		}
	}

	for _, diagnostic := range diagnostics {
		message := diagnosticMessage(diagnostic)
		matched := false
		for _, resource := range resources {
			if mentionsResourceType(message, resource.Type) {
				findings = append(findings, schema.Deprecation{
					Address: resource.Address, Type: resource.Type, Source: "terraform-provider",
					Message: message, Filename: diagnostic.Filename, Line: diagnostic.Line,
				})
				matched = true
			}
		}
		if !matched {
			findings = append(findings, schema.Deprecation{
				Source: "terraform-provider", Message: message,
				Filename: diagnostic.Filename, Line: diagnostic.Line,
			})
		}
	}
	return findings, nil
}

func loadRules(path string) (rulesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return rulesFile{}, fmt.Errorf("read deprecation rules %s: %w", path, err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var file rulesFile
	if err := decoder.Decode(&file); err != nil {
		return rulesFile{}, fmt.Errorf("parse deprecation rules %s: %w", path, err)
	}
	for resourceType, entry := range file.Resources {
		if strings.TrimSpace(resourceType) == "" || strings.TrimSpace(entry.Message) == "" {
			return rulesFile{}, fmt.Errorf("deprecation rules %s: each resource needs a non-empty type and message", path)
		}
		if entry.Reference != "" {
			u, err := url.ParseRequestURI(entry.Reference)
			if err != nil || u.Scheme == "" || u.Host == "" {
				return rulesFile{}, fmt.Errorf("deprecation rules %s: invalid reference URL for %s", path, resourceType)
			}
		}
	}
	return file, nil
}

func mentionsResourceType(message, resourceType string) bool {
	message = strings.ToLower(message)
	resourceType = strings.ToLower(resourceType)
	for offset := 0; offset < len(message); {
		relative := strings.Index(message[offset:], resourceType)
		if relative < 0 {
			return false
		}
		start := offset + relative
		end := start + len(resourceType)
		if (start == 0 || !identifierByte(message[start-1])) && (end == len(message) || !identifierByte(message[end])) {
			return true
		}
		offset = start + 1
	}
	return false
}

func identifierByte(b byte) bool {
	return b == '_' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}

func diagnosticMessage(d plan.TerraformDeprecation) string {
	summary := strings.TrimSpace(d.Summary)
	detail := strings.TrimSpace(d.Detail)
	if detail == "" || strings.EqualFold(summary, detail) {
		return summary
	}
	return summary + ": " + detail
}
