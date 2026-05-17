package server

import (
	"net/http"
	"time"
)

// handleTheme is the no-JS fallback for the theme switcher. JS users
// short-circuit this entirely (see layout.gohtml — the form's submit is
// preventDefault'd and the cookie + body class are updated client-side).
//
// HttpOnly is intentionally NOT set so the JS path can write the same
// cookie. Accent preference is non-sensitive — no need to protect from
// document.cookie access.
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
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
	})

	// Send the no-JS user back to the page they were on so the new accent
	// renders without losing their place.
	ref := r.Header.Get("Referer")
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}
