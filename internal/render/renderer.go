package render

import (
	"fmt"
	"io"

	"github.com/osifotos/price-checker/pkg/schema"
)

// Options controls presentation. They do not change any value in a document.
type Options struct {
	Period         schema.Period
	ShowComponents bool
	Color          bool
}

// Renderer writes a breakdown.
type Renderer interface {
	Render(w io.Writer, b *schema.Breakdown, opts Options) error
}

// DiffRenderer writes a diff.
type DiffRenderer interface {
	RenderDiff(w io.Writer, d *schema.DiffResult, opts Options) error
}

// Format names accepted by RendererFor.
const (
	FormatTable   = "table"
	FormatJSON    = "json"
	FormatHTML    = "html"
	FormatComment = "github-comment"
)

// RendererFor resolves a --format value to its renderer pair. Every built-in
// format implements both interfaces.
func RendererFor(format string) (Renderer, DiffRenderer, error) {
	switch format {
	case "", FormatTable:
		r := tableRenderer{}
		return r, r, nil
	case FormatJSON:
		r := jsonRenderer{}
		return r, r, nil
	case FormatHTML:
		r := htmlRenderer{}
		return r, r, nil
	case FormatComment:
		r := ghRenderer{}
		return r, r, nil
	default:
		return nil, nil, fmt.Errorf("unknown format %q (want table, json, html, or github-comment)", format)
	}
}
