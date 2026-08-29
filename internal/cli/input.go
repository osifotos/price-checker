package cli

import (
	"os"
	"strings"
)

// InputKind classifies a --path / --compare-to value.
type InputKind int

const (
	InputStdin InputKind = iota
	InputTerraformDir
	InputPlanFile
	InputPriorJSON
)

// Classify decides how to consume value.
func Classify(value string) (InputKind, error) {
	if value == "" || value == "-" {
		return InputStdin, nil
	}
	info, err := os.Stat(value)
	if err != nil {
		return InputPlanFile, err
	}
	if info.IsDir() {
		return InputTerraformDir, nil
	}
	b, err := os.ReadFile(value)
	if err != nil {
		return InputPlanFile, err
	}
	head := b
	if len(head) > 4096 {
		head = head[:4096]
	}
	s := string(head)
	switch {
	case strings.Contains(s, `"format_version"`):
		return InputPlanFile, nil
	case strings.Contains(s, `"schema_version"`):
		return InputPriorJSON, nil
	default:
		// let the plan parser produce a precise error
		return InputPlanFile, nil
	}
}
