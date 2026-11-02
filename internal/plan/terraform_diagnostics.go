package plan

import (
	"bufio"
	"bytes"
	"encoding/json"
	"sort"
	"strings"
)

type TerraformDeprecation struct {
	Summary  string
	Detail   string
	Filename string
	Line     int
}

type terraformUIEvent struct {
	Type       string `json:"type"`
	Diagnostic struct {
		Severity string `json:"severity"`
		Summary  string `json:"summary"`
		Detail   string `json:"detail"`
		Range    *struct {
			Filename string `json:"filename"`
			Start    struct {
				Line int `json:"line"`
			} `json:"start"`
		} `json:"range"`
	} `json:"diagnostic"`
}

func terraformDeprecations(raw []byte) []TerraformDeprecation {
	var out []TerraformDeprecation
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		var event terraformUIEvent
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Type != "diagnostic" {
			continue
		}
		if event.Diagnostic.Severity != "warning" && event.Diagnostic.Severity != "warn" {
			continue
		}
		if !strings.Contains(strings.ToLower(event.Diagnostic.Summary+" "+event.Diagnostic.Detail), "deprecat") {
			continue
		}
		warning := TerraformDeprecation{
			Summary: event.Diagnostic.Summary,
			Detail:  event.Diagnostic.Detail,
		}
		if event.Diagnostic.Range != nil {
			warning.Filename = event.Diagnostic.Range.Filename
			warning.Line = event.Diagnostic.Range.Start.Line
		}
		out = append(out, warning)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Filename != out[j].Filename {
			return out[i].Filename < out[j].Filename
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Summary < out[j].Summary
	})
	return out
}
