package render

import (
	_ "embed"
	"html/template"
	"io"

	"github.com/example/price-checker/pkg/schema"
)

//go:embed html/report.html.tmpl
var htmlTemplateSrc string

//go:embed html/report.css
var htmlCSS string

//go:embed html/report.js
var htmlJS string

var reportTmpl = template.Must(template.New("report").Parse(htmlTemplateSrc))

type htmlRenderer struct{}

// htmlPage is htmlVM plus the inlined assets, marked safe for their contexts.
type htmlPage struct {
	htmlVM
	CSS template.CSS
	JS  template.JS
}

func (htmlRenderer) Render(w io.Writer, b *schema.Breakdown, _ Options) error {
	return reportTmpl.Execute(w, htmlPage{
		htmlVM: breakdownVM(b),
		CSS:    template.CSS(htmlCSS),
		JS:     template.JS(htmlJS),
	})
}

func (htmlRenderer) RenderDiff(w io.Writer, d *schema.DiffResult, _ Options) error {
	return reportTmpl.Execute(w, htmlPage{
		htmlVM: diffVM(d),
		CSS:    template.CSS(htmlCSS),
		JS:     template.JS(htmlJS),
	})
}
