package server

import "net/http"

func (s *server) handleSavedList(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "page_saved.gohtml", s.basePageData(r))
}

func (s *server) handleSavedView(w http.ResponseWriter, r *http.Request) {
	data := s.basePageData(r)
	data.Slug = r.PathValue("slug")
	s.renderPage(w, r, "page_saved_view.gohtml", data)
}

func (s *server) handleSavedCreate(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "saved/create: not implemented", http.StatusNotImplemented)
}

func (s *server) handleSavedDelete(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "saved/delete: not implemented", http.StatusNotImplemented)
}
