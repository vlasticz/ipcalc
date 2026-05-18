package server

import (
	"net/http"
	"strings"

	"github.com/vlasticz/ipcalc/internal/ipcalc"
	"github.com/vlasticz/ipcalc/internal/storage"
)

// savedResult is the small ack rendered into the save form's slot after
// POST /saved succeeds — slug for linking, label echoed back.
type savedResult struct {
	Slug  string
	Label string
}

// savedListItem wraps a Saved row with the normalised network for display.
// Stored input_json preserves what the user typed; the listing column
// shows the canonical "network/prefix" form computed at render time.
type savedListItem struct {
	*storage.Saved
	Network string
}

// pageData is the canonical render context for full pages. Layout reads
// .Accent, .Accents, and .ActivePage; pages read their own fields.
type pageData struct {
	Accent     string
	Accents    []string
	ActivePage string // "calc" | "split" | "saved" — drives nav highlight

	// Calculator
	IP, Mask, Warning string
	Result            *ipcalc.Result

	// Saved view / list
	Slug       string
	Label      string
	SavedList  []savedListItem
	SavedItem  *storage.Saved
	SaveResult *savedResult // populated after a successful save

	// Split
	SplitMode      string // "equal" or "vlsm" — drives which partial wrapper id is used
	SplitParent    string // form value, echoed back on error
	SplitN         string // form value (Equal mode)
	SplitHosts     string // form value (VLSM mode), comma-separated
	SplitResults   []ipcalc.Result
	SplitRequested []int  // for VLSM, aligned with SplitResults so labels line up
	SplitError     string
}

func (s *server) basePageData(r *http.Request) pageData {
	return pageData{
		Accent:     accentFromContext(r.Context()),
		Accents:    allAccents,
		ActivePage: activePageFromPath(r.URL.Path),
	}
}

// activePageFromPath maps a request path to the top-level nav section.
func activePageFromPath(p string) string {
	switch {
	case p == "/" || p == "":
		return "calc"
	case strings.HasPrefix(p, "/split"):
		return "split"
	case strings.HasPrefix(p, "/saved"):
		return "saved"
	}
	return ""
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
