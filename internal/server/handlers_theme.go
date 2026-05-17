package server

import (
	"net/http"
	"time"
)

// handleTheme sets the accent cookie. Returns 204 plus HX-Refresh so HTMX
// reloads the page with the new accent applied server-side (no FOUC).
func (s *server) handleTheme(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	v := r.PostFormValue("accent")
	if !isValidAccent(v) {
		http.Error(w, "invalid accent", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     accentCookieName,
		Value:    v,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 365, // 1y
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
	})
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusNoContent)
}
