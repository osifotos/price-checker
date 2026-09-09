package render

import (
	"encoding/json"
	"io"

	"github.com/osifotos/price-checker/pkg/schema"
)

type jsonRenderer struct{}

func (jsonRenderer) Render(w io.Writer, b *schema.Breakdown, _ Options) error {
	return encodeJSON(w, b)
}

func (jsonRenderer) RenderDiff(w io.Writer, d *schema.DiffResult, _ Options) error {
	return encodeJSON(w, d)
}

func encodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
