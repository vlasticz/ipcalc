// Package server wires HTTP routes for the ipcalc UI. It translates HTTP
// into calls against internal/ipcalc and internal/storage, and renders via
// internal/templates. No business logic lives here.
package server

import (
	"log/slog"
	"net/http"

	"github.com/vlasticz/ipcalc/internal/storage"
	"github.com/vlasticz/ipcalc/internal/templates"
)

// Deps is the wiring dependencies handed to New.
type Deps struct {
	Logger    *slog.Logger
	Templates *templates.Templates
	Storage   *storage.Storage
	StaticDir string
}

type server struct {
	deps Deps
}

// New builds the HTTP handler tree.
func New(d Deps) http.Handler {
	s := &server{deps: d}
	mux := http.NewServeMux()

	// Liveness — defined here so middleware logging skips it cleanly.
	mux.HandleFunc("GET /healthz", s.handleHealth)

	// Static assets (Tailwind output, htmx.min.js, favicon).
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(d.StaticDir))))

	// Calculator
	mux.HandleFunc("GET /{$}", s.handleCalcPage)
	mux.HandleFunc("POST /calc", s.handleCalcFragment)

	// Split (stubs)
	mux.HandleFunc("GET /split", s.handleSplitPage)
	mux.HandleFunc("POST /split/equal", s.handleSplitEqual)
	mux.HandleFunc("POST /split/vlsm", s.handleSplitVLSM)

	// Saved (stubs)
	mux.HandleFunc("GET /saved", s.handleSavedList)
	mux.HandleFunc("POST /saved", s.handleSavedCreate)
	mux.HandleFunc("GET /saved/{slug}", s.handleSavedView)
	mux.HandleFunc("DELETE /saved/{slug}", s.handleSavedDelete)

	// Theme switcher
	mux.HandleFunc("POST /theme", s.handleTheme)

	return s.wrapMiddleware(mux)
}
