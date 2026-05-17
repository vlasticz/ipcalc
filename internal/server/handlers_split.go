package server

import "net/http"

func (s *server) handleSplitPage(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "page_split.gohtml", s.basePageData(r))
}

func (s *server) handleSplitEqual(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "split/equal: not implemented", http.StatusNotImplemented)
}

func (s *server) handleSplitVLSM(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "split/vlsm: not implemented", http.StatusNotImplemented)
}
