// Package templates owns the html/template setup: embedded .gohtml files,
// one parse tree per page (so each page can override layout blocks), and a
// separate tree for HTMX partials.
//
// Templates are parsed once at startup. Hot-reload during development is
// handled by Air restarting the process on .gohtml changes.
package templates

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
)

//go:embed *.gohtml
var fs embed.FS

// Templates is the rendering façade used by HTTP handlers.
type Templates struct {
	pages    map[string]*template.Template
	partials *template.Template
}

// Parse loads and parses every .gohtml file at startup.
func Parse() (*Templates, error) {
	entries, err := fs.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read embed: %w", err)
	}

	var layoutAndPartials []string
	var pages []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch {
		case e.Name() == "layout.gohtml":
			layoutAndPartials = append(layoutAndPartials, e.Name())
		case strings.HasPrefix(e.Name(), "partial_"):
			layoutAndPartials = append(layoutAndPartials, e.Name())
		case strings.HasPrefix(e.Name(), "page_"):
			pages = append(pages, e.Name())
		}
	}

	if len(layoutAndPartials) == 0 {
		return nil, fmt.Errorf("no layout/partial templates found")
	}

	base, err := template.New("").Funcs(funcs).ParseFS(fs, layoutAndPartials...)
	if err != nil {
		return nil, fmt.Errorf("parse base: %w", err)
	}

	t := &Templates{
		pages:    make(map[string]*template.Template, len(pages)),
		partials: base,
	}
	for _, p := range pages {
		clone, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone base for %s: %w", p, err)
		}
		if _, err := clone.ParseFS(fs, p); err != nil {
			return nil, fmt.Errorf("parse page %s: %w", p, err)
		}
		t.pages[p] = clone
	}
	return t, nil
}

// RenderPage executes the named page through layout. `page` is the bare
// filename, e.g. "page_calc.gohtml".
func (t *Templates) RenderPage(w io.Writer, page string, data any) error {
	tpl, ok := t.pages[page]
	if !ok {
		return fmt.Errorf("unknown page: %s", page)
	}
	return tpl.ExecuteTemplate(w, "layout", data)
}

// RenderPartial renders a named partial (HTMX fragment) without layout.
func (t *Templates) RenderPartial(w io.Writer, name string, data any) error {
	return t.partials.ExecuteTemplate(w, name, data)
}

// funcs is the template FuncMap.
var funcs = template.FuncMap{
	"add": func(a, b int) int { return a + b },
}
