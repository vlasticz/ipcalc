package server

import (
	"net/http"

	"github.com/vlasticz/ipcalc/internal/ipcalc"
)

// pageData is the canonical render context for full pages. Layout reads
// .Accent and .Accents; pages read their own fields.
type pageData struct {
	Accent  string
	Accents []string

	// Calculator
	IP, Mask, Warning string
	Result            *ipcalc.Result

	// Saved view
	Slug  string
	Label string
}

func (s *server) basePageData(r *http.Request) pageData {
	return pageData{
		Accent:  accentFromContext(r.Context()),
		Accents: allAccents,
	}
}

func (s *server) renderPage(w http.ResponseWriter, _ *http.Request, page string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.deps.Templates.RenderPage(w, page, data); err != nil {
		s.deps.Logger.Error("render page", "page", page, "err", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (s *server) renderPartial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.deps.Templates.RenderPartial(w, name, data); err != nil {
		s.deps.Logger.Error("render partial", "partial", name, "err", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
